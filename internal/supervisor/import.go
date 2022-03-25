// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/events"

	awat "github.com/mattermost/awat/model"
	toolsAWS "github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// AWATClient is the programmatic interface to the AWAT API.
type AWATClient interface {
	CreateTranslation(translationRequest *awat.TranslationRequest) (*awat.TranslationStatus, error)
	GetTranslationStatus(translationID string) (*awat.TranslationStatus, error)
	GetTranslationStatusesByInstallation(installationID string) ([]*awat.TranslationStatus, error)
	GetAllTranslations() ([]*awat.TranslationStatus, error)

	GetTranslationReadyToImport(request *awat.ImportWorkRequest) (*awat.ImportStatus, error)
	GetImportStatusesByInstallation(installationID string) ([]*awat.ImportStatus, error)
	GetImportStatusesByTranslation(translationID string) ([]*awat.ImportStatus, error)
	ListImports() ([]*awat.ImportStatus, error)
	GetImportStatus(importID string) (*awat.ImportStatus, error)

	CompleteImport(completed *awat.ImportCompletedWorkRequest) error
	ReleaseLockOnImport(importID string) error
}

// ImportSupervisor is a supervisor which performs Workspace Imports
// from ready Imports produced by the AWAT. It periodically queries
// the AWAT for Imports waiting to be performed and then performs
// imports serially
type ImportSupervisor struct {
	awsClient      toolsAWS.AWS
	awatClient     AWATClient
	logger         logrus.FieldLogger
	store          importStore
	provisioner    importProvisioner
	eventsProducer eventProducer
	ID             string
}

type importStore interface {
	GetInstallations(filter *model.InstallationFilter, includeGroupConfig, includeGroupConfigOverrides bool) ([]*model.Installation, error)

	installationStore
}

type importProvisioner interface {
	ExecClusterInstallationCLI(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error)
}

type mmctl struct {
	cluster             *model.Cluster
	clusterInstallation *model.ClusterInstallation
	installation        *model.Installation

	*ImportSupervisor
}

func (m *mmctl) Run(args ...string) ([]byte, error) {
	args = append([]string{"mmctl", "--format", "json", "--local"}, args...)
	return m.provisioner.ExecClusterInstallationCLI(m.cluster, m.clusterInstallation, args...)
}

type team struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type jobResponse struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Progress int    `json:"progress"`

	jobResponseData `json:"data"`
}

type jobResponseData struct {
	Error      string `json:"error"`
	LineNumber string `json:"line_number"`
	ImportFile string `json:"import_file"`
}

// NewImportSupervisor creates a new Import Supervisor
func NewImportSupervisor(awsClient toolsAWS.AWS, awat AWATClient, store importStore, provisioner importProvisioner, eventsProducer eventProducer, logger logrus.FieldLogger) *ImportSupervisor {
	return &ImportSupervisor{
		awsClient:      awsClient,
		awatClient:     awat,
		store:          store,
		logger:         logger,
		provisioner:    provisioner,
		eventsProducer: eventsProducer,

		// TODO replace this with the Pod ID from env var
		ID: model.NewID(),
	}
}

// Do checks to see if there is an Import that is ready to be
// imported, and if so, does that. Otherwise, it does nothing.
func (s *ImportSupervisor) Do() error {
	err := s.completeImports()
	if err != nil {
		s.logger.WithError(err).Warn("Failed to move installations with completed imports back to state stable")
	}

	work, err := s.awatClient.GetTranslationReadyToImport(
		&awat.ImportWorkRequest{
			ProvisionerID: s.ID,
		})
	if err != nil {
		return errors.Wrap(err, "failed to get a ready Import from the AWAT")
	}
	if work == nil {
		// nothing to do
		return nil
	}

	defer func() {
		err = s.awatClient.ReleaseLockOnImport(work.ID)
		if err != nil {
			s.logger.WithError(err).Warnf("Failed to release lock on Import %s", work.ID)
		}
	}()

	err = s.importTranslation(work)
	if err != nil {
		s.logger.WithError(err).Errorf("Failed to perform work on Import %s", work.ID)
		workError := err.Error()
		go func() {
			for {
				err := s.awatClient.CompleteImport(
					&awat.ImportCompletedWorkRequest{
						ID:         work.ID,
						CompleteAt: model.GetMillis(),
						Error:      workError,
					})
				if err == nil {
					return
				}
				s.logger.WithError(err).Errorf("failed to report error to AWAT for Import %s", work.ID)
				time.Sleep(time.Second * 5)
			}
		}()
	}

	return err
}

// Shutdown is called when the ImportSupervisor is stopped.
// TODO change Shutdown from a no-op to allowing it to unlock ongoing
// Imports so that they are detected as interrupted by the AWAT and
// can be re-served to another Provisioner instance as a cleaner retry
// pattern
func (s *ImportSupervisor) Shutdown() {
	s.logger.Debug("Shutting down import supervisor")
}

func (s *ImportSupervisor) importTranslation(imprt *awat.ImportStatus) error {
	logger := s.logger.WithFields(logrus.Fields{
		"import":       imprt.ID,
		"installation": imprt.InstallationID,
	})
	logger.Info("Starting installation import")

	installation, err := s.store.GetInstallation(imprt.InstallationID, false, false)
	if err != nil {
		return errors.Wrapf(err, "failed to look up Installation %s", imprt.InstallationID)
	}
	if installation == nil {
		return errors.Errorf("Installation %s not found for Import %s", imprt.InstallationID, imprt.ID)
	}

	// grab Installation lock
	lock := newInstallationLock(installation.ID, s.ID, s.store, s.logger)
	if !lock.TryLock() {
		return errors.Wrapf(err, "failed to get lock on Installation %s", installation.ID)
	}
	defer lock.Unlock()

	// check Installation state is valid
	if installation.State != model.InstallationStateStable {
		if installation.State == model.InstallationStateDeleted {
			err = s.awatClient.CompleteImport(
				&awat.ImportCompletedWorkRequest{
					ID:         imprt.ID,
					CompleteAt: model.GetMillis(),
					Error:      "installation was deleted",
				})
			if err != nil {
				s.logger.WithError(err).Warnf("failed to cancel Import %s for deleted Installation %s",
					imprt.ID, installation.ID)
			}
			return errors.Errorf("import for Installation %s cannot continue as the Installation was deleted", installation.ID)
		}
		return errors.Errorf("skipping import on Installation %s with state %s. State must be stable to begin work.", installation.ID, installation.State)
	}

	// mark this Installation as import-in-progress
	installation.State = model.InstallationStateImportInProgress
	err = s.store.UpdateInstallation(installation)
	if err != nil {
		return errors.Wrapf(err, "failed to mark Installation %s as %s", installation.ID, model.InstallationStateImportInProgress)
	}
	err = s.eventsProducer.ProduceInstallationStateChangeEvent(installation, model.InstallationStateStable, importEventExtraData(imprt)...)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create installation state change event")
	}

	defer func() {
		installation.State = model.InstallationStateImportComplete
		err = s.store.UpdateInstallation(installation)
		if err != nil {
			s.logger.WithError(err).Errorf("Failed to mark Installation %s as state stable", installation.ID)
			return
		}
		err = s.eventsProducer.ProduceInstallationStateChangeEvent(installation, model.InstallationStateImportInProgress, importEventExtraData(imprt)...)
		if err != nil {
			s.logger.WithError(err).Error("Failed to create installation state change event")
		}
	}()

	// find the CI and Cluster the Installation is on for executing commands
	clusterInstallations, err := s.store.GetClusterInstallations(
		&model.ClusterInstallationFilter{
			Paging:         model.AllPagesNotDeleted(),
			InstallationID: installation.ID,
		})
	if err != nil {
		return errors.Wrap(err, "failed to lookup cluster installation for cluster")
	}

	if len(clusterInstallations) < 1 {
		return errors.Errorf("no ClusterInstallations found for Installation %s", installation.ID)
	}
	ci := clusterInstallations[0]

	cluster, err := s.store.GetCluster(ci.ClusterID)
	if err != nil {
		return errors.Wrap(err, "failed to lookup cluster for cluster installation")
	}

	// ensure that the Installation will be able to accept the import
	mmctl := &mmctl{cluster: cluster, clusterInstallation: ci, ImportSupervisor: s}
	err = s.ensureTeamSettings(mmctl, imprt)
	if err != nil {
		return errors.Wrap(err, "failed to ensure team settings")
	}

	srcKey, err := s.copyImportToWorkspaceFilestore(imprt, installation, logger)
	if err != nil {
		return errors.Wrapf(err, "failed to copy workspace import archive to Installation %s filestore", installation.ID)
	}

	err = s.startImportProcessAndWait(mmctl, logger, srcKey, imprt.ID)
	if err != nil {
		return errors.Wrap(err, "failed to complete import process")
	}

	logger.Info("Installation import job complete")

	return nil
}

func (s *ImportSupervisor) teamAlreadyExists(mmctl *mmctl, teamName string) (bool, error) {
	output, err := mmctl.Run("team", "search", teamName)
	if err != nil {
		return false, errors.Wrapf(err, "failed to search for team %s", teamName)
	}
	teamSearch := []*team{}
	_ = json.Unmarshal(output, &teamSearch)

	for _, team := range teamSearch {
		if team.Name == teamName {
			return true, nil
		}
	}
	return false, nil
}

func (s *ImportSupervisor) ensureTeamSettings(mmctl *mmctl, imprt *awat.ImportStatus) error {
	if imprt.Type != awat.MattermostWorkspaceBackupType {
		// ensure that the team exists
		found, err := s.teamAlreadyExists(mmctl, imprt.Team)
		if err != nil {
			return errors.Wrapf(err, "failed to determine if Team %s already exists in workspace", imprt.Team)
		}

		// if the team doesn't exist, create it
		if !found {
			output, err := mmctl.Run("team", "create", "--name", imprt.Team, "--display_name", imprt.Team)
			if err != nil {
				return errors.Wrapf(err, "failed to find or create team %s; full output was:%s\n", imprt.Team, string(output))
			}
		}
	}

	// ensure that there will be enough new user slots for this import

	output, err := mmctl.Run("config", "get", "TeamSettings.MaxUsersPerTeam")
	if err != nil {
		return errors.Wrapf(err, "failed to get max user limit")
	}

	// despite requesting JSON mmctl always just returns a bare number
	// for these config commands
	currentMaxUsers, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return errors.Wrapf(err, "failed to convert \"%s\" to a number", output)
	}

	maxUsers := currentMaxUsers + imprt.Users
	_, err = mmctl.Run("config", "set", "TeamSettings.MaxUsersPerTeam", strconv.Itoa(maxUsers))
	if err != nil {
		return errors.Wrapf(err, "failed to add %d users to MaxUsersPerTeam", imprt.Users)
	}

	return nil
}

func (s *ImportSupervisor) getBucketForInstallation(installation *model.Installation) (string, error) {
	switch installation.Filestore {
	// TODO handle single tenant bucket names
	case model.InstallationFilestoreMultiTenantAwsS3, model.InstallationFilestoreBifrost:
		return s.awsClient.GetMultitenantBucketNameForInstallation(installation.ID, s.store)
	case model.InstallationFilestoreAwsS3, model.InstallationFilestoreMinioOperator:
		return "", errors.Errorf("support for workspace imports to workspaces with the filestore type %s not yet supported", installation.Filestore)
	default:
		return "", errors.Errorf("encountered unknown filestore type %s", installation.Filestore)
	}
}

// copyImportToWorkspaceFilestorecopies the archive from the AWAT's bucket to the Installation's
// bucket. Returns the key of the archive in the source bucket or an
// error
func (s *ImportSupervisor) copyImportToWorkspaceFilestore(imprt *awat.ImportStatus, installation *model.Installation, logger logrus.FieldLogger) (string, error) {
	// calculate necessary paths and copy the archive to the Installation's S3 bucket
	source := strings.SplitN(imprt.Resource, "/", 2)
	if len(source) != 2 {
		return "", errors.Errorf("failed to parse bucket/key from Import %s Resource %s", imprt.ID, imprt.Resource)
	}
	srcBucket := source[0]
	srcKey := source[1]
	destKey := fmt.Sprintf("%s/import/%s", installation.ID, srcKey)

	destBucket, err := s.getBucketForInstallation(installation)
	if err != nil {
		return "", errors.Wrapf(err, "failed to determine bucket name for Installation %s", installation.ID)
	}

	logger.Debugf("Copying %s/%s to %s/%s", srcBucket, srcKey, destBucket, destKey)
	err = s.awsClient.S3LargeCopy(&srcBucket, &srcKey, &destBucket, &destKey)
	if err != nil {
		return "", errors.Wrapf(err, "failed to copy archive to Installation %s", installation.ID)
	}

	return srcKey, nil
}

func (s *ImportSupervisor) startImportProcessAndWait(mmctl *mmctl, logger logrus.FieldLogger, importArchiveFilename, awatImportID string) error {
	output, err := mmctl.Run("import", "process", importArchiveFilename)
	if err != nil {
		return errors.Wrap(err, "failed to start import process in Mattermost itself")
	}

	jobResponses := []*jobResponse{}
	err = json.Unmarshal([]byte(output), &jobResponses)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal Job response from Mattermost")
	}

	if len(jobResponses) < 1 {
		return errors.New("response was empty")
	}

	jobID := jobResponses[0].ID
	logger.Infof("Started Import job %s", jobID)
	return s.waitForImportToComplete(mmctl, logger, jobID, awatImportID)
}

func (s *ImportSupervisor) waitForImportToComplete(mmctl *mmctl, logger logrus.FieldLogger, mattermostJobID, awatImportID string) error {
	complete := false
	var (
		resp         *jobResponse
		output       []byte
		jobResponses []*jobResponse
	)

	for !complete {
		output, err := mmctl.Run("import", "job", "show", mattermostJobID)
		if err != nil {
			logger.WithError(err).Warn("failed to check job")
			time.Sleep(5 * time.Second)
			continue
		}

		err = json.Unmarshal([]byte(output), &jobResponses)
		if err != nil {
			logger.WithError(err).Warn("failed to check job; bad JSON")
			time.Sleep(5 * time.Second)
			continue
		}
		if len(jobResponses) != 1 {
			return errors.Errorf("unexpected number of responses from jobs API endpoint (%d)", len(jobResponses))
		}
		resp = jobResponses[0]
		if resp.Status != "pending" && resp.Status != "in_progress" {
			complete = true
		}
		if resp.Status == "error" {
			errorString := fmt.Sprintf("import job failed with error %s", resp.Error)
			if resp.LineNumber != "" {
				errorString = fmt.Sprintf("%s on line %s", errorString, resp.LineNumber)
			}
			if resp.ImportFile != "" {
				errorString = fmt.Sprintf("%s in JSONL file from %s", errorString, resp.ImportFile)
			}
			return errors.New(errorString)
		}

		logger.Debugf("Waiting for job %s to complete. Status: %s Progress: %d", resp.ID, resp.Status, resp.Progress)
		time.Sleep(5 * time.Second)
	}

	logger.Infof("Import Job %s successfully completed", resp.ID)
	logger.Debugf("Completed with output: %s", output)

	return nil
}

// completeImports transitions Installations back to stable after
// checking with the AWAT to make sure that it has detected the
// completion of the import
func (s *ImportSupervisor) completeImports() error {
	installationList, err := s.store.GetInstallations(
		&model.InstallationFilter{
			Paging: model.AllPagesNotDeleted(),
			State:  model.InstallationStateImportComplete,
		}, false, false)
	if err != nil {
		return errors.Wrap(err, "failed to list Installations")
	}

	for _, installation := range installationList {
		err = s.checkInstallation(installation)
		if err != nil {
			s.logger.WithError(err).Warnf("failed to check to see if Installation %s has completed its import", installation.ID)
			continue
		}
	}

	return nil
}

func (s *ImportSupervisor) checkInstallation(installation *model.Installation) error {
	importStatusList, err := s.awatClient.GetImportStatusesByInstallation(installation.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to get Import status for Installation %s", installation.ID)
	}
	if len(importStatusList) == 0 {
		return errors.Wrapf(err, "no imports found for Installation with state %s, %s", installation.State, installation.ID)
	}
	// find the most recently completed Import; that's the only one we
	// care about
	var mostRecentImport *awat.ImportStatus
	for _, is := range importStatusList {
		if is.State != awat.ImportStateComplete {
			// An import is still running against this Installation.
			// Multiple parallel imports against an Installation are not supported.
			break
		}
		if mostRecentImport == nil || is.CompleteAt > mostRecentImport.CompleteAt {
			mostRecentImport = is
		}
	}
	if mostRecentImport == nil {
		// an Import might still be running
		return nil
	}

	locked, err := s.store.LockInstallation(installation.ID, s.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to lock Installation %s", installation.ID)
	}
	if !locked {
		return errors.Errorf("failed to lock Installation %s", installation.ID)
	}
	defer func() {
		unlocked, err := s.store.UnlockInstallation(installation.ID, s.ID, false)
		if err != nil {
			s.logger.WithError(err).Warnf("failed to unlock Installation %s to mark Import %s complete", installation.ID, mostRecentImport.ID)
			return
		}
		if !unlocked {
			s.logger.Warnf("failed to unlock Installation %s to mark Import %s complete", installation.ID, mostRecentImport.ID)
		}
	}()

	installation, err = s.store.GetInstallation(installation.ID, false, false)
	if err != nil {
		return errors.Wrapf(err, "failed to get Installation %s after locking", installation.ID)
	}

	// no Imports are running, the Installation may be moved to stable
	installation.State = model.InstallationStateStable
	err = s.store.UpdateInstallation(installation)
	if err != nil {
		return errors.Wrap(err, "failed to mark Installation stable")
	}

	err = s.eventsProducer.ProduceInstallationStateChangeEvent(installation, model.InstallationStateImportComplete, importEventExtraData(mostRecentImport)...)
	if err != nil {
		s.logger.WithError(err).Error("Failed to create installation state change event")
	}

	return nil
}

func importEventExtraData(importStatus *awat.ImportStatus) []events.DataField {
	return []events.DataField{
		{Key: "TranslationID", Value: importStatus.TranslationID},
		{Key: "ImportID", Value: importStatus.ID},
	}
}
