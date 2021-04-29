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

	awat "github.com/mattermost/awat/model"
	toolsAWS "github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// ImportSupervisor is a supervisor which performs Workspace Imports
// from ready Imports produced by the AWAT. It periodically queries
// the AWAT for Imports waiting to be performed and then performs
// imports serially
type ImportSupervisor struct {
	awsClient   toolsAWS.AWS
	awatClient  awat.Client
	logger      logrus.FieldLogger
	store       installationStore
	provisioner importProvisioner
	ID          string
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

	jobResponseData
}

type jobResponseData struct {
	Error      string `json:"error"`
	LineNumber string `json:"line_number"`
}

// NewImportSupervisor creates a new Import Supervisor
func NewImportSupervisor(awsClient toolsAWS.AWS, awat awat.Client, store installationStore, provisioner importProvisioner, logger logrus.FieldLogger) *ImportSupervisor {
	return &ImportSupervisor{
		awsClient:   awsClient,
		awatClient:  awat,
		store:       store,
		logger:      logger,
		provisioner: provisioner,

		// TODO replace this with the Pod ID from env var
		ID: model.NewID(),
	}
}

// Do checks to see if there is an Import that is ready to be
// imported, and if so, does that. Otherwise, it does nothing.
func (is *ImportSupervisor) Do() error {
	work, err := is.awatClient.GetTranslationReadyToImport(
		&awat.ImportWorkRequest{
			ProvisionerID: is.ID,
		})
	if err != nil {
		return errors.Wrap(err, "failed to get a ready Import from the AWAT")
	}
	if work == nil {
		// nothing to do
		return nil
	}
	defer func() {
		err = is.awatClient.ReleaseLockOnImport(work.ID)
		if err != nil {
			is.logger.WithError(err).Warnf("failed to release lock on Import %s", work.ID)
		}
	}()

	err = is.importTranslation(work)
	if err != nil {
		is.logger.WithError(err).Errorf("failed to perform work on Import %s", work.ID)
	}

	return err
}

// Shutdown is called when the ImportSupervisor is stopped.
// TODO change Shutdown from a no-op to allowing it to unlock ongoing
// Imports so that they are detected as interrupted by the AWAT and
// can be re-served to another Provisioner instance as a cleaner retry
// pattern
func (is *ImportSupervisor) Shutdown() {
	is.logger.Debug("Shutting down import supervisor")
}

func (is *ImportSupervisor) importTranslation(imprt *awat.ImportStatus) error {
	// get installation metadata
	installation, err := is.store.GetInstallation(imprt.InstallationID, false, false)
	if err != nil {
		return errors.Wrapf(err, "failed to look up Installation %s", imprt.InstallationID)
	}
	if installation == nil {
		return errors.Errorf("Installation %s not found for Import %s", imprt.InstallationID, imprt.ID)
	}

	// grab Installation lock
	lock := newInstallationLock(installation.ID, is.ID, is.store, is.logger)
	if !lock.TryLock() {
		return errors.Wrapf(err, "failed to get lock on Installation %s", installation.ID)
	}
	defer lock.Unlock()

	// check Installation state is valid
	if installation.State != model.InstallationStateStable {
		return errors.Errorf("Skipping import on Installation %s with state %s. State must be stable to begin work.", installation.ID, installation.State)
	}

	// mark this Installation as import-in-progress
	installation.State = model.InstallationStateImportInProgress
	err = is.store.UpdateInstallation(installation)
	if err != nil {
		return errors.Wrapf(err, "failed to mark Installation %s as %s", installation.ID, model.InstallationStateImportInProgress)
	}
	err = webhook.SendToAllWebhooks(is.store, &model.WebhookPayload{
		Type:      model.TypeInstallation,
		ID:        installation.ID,
		NewState:  model.InstallationStateImportInProgress,
		OldState:  model.InstallationStateStable,
		ExtraData: map[string]string{"TranslationID": imprt.TranslationID, "ImportID": imprt.ID},
	}, is.logger.WithField("webhookEvent", model.InstallationStateImportInProgress))
	if err != nil {
		is.logger.WithError(err).Errorf("failed to send webhooks")
	}

	defer func() {
		installation.State = model.InstallationStateStable
		err = is.store.UpdateInstallation(installation)
		if err != nil {
			is.logger.WithError(err).Errorf("failed to mark Installation %s as state stable", installation.ID)
			return
		}
		err = webhook.SendToAllWebhooks(is.store, &model.WebhookPayload{
			Type:      model.TypeInstallation,
			ID:        installation.ID,
			NewState:  model.InstallationStateStable,
			OldState:  model.InstallationStateImportInProgress,
			ExtraData: map[string]string{"TranslationID": imprt.TranslationID, "ImportID": imprt.ID},
		}, is.logger.WithField("webhookEvent", model.InstallationStateImportInProgress))
		if err != nil {
			is.logger.WithError(err).Errorf("failed to send webhooks")
		}
	}()

	// find the CI and Cluster the Installation is on for executing commands
	clusterInstallations, err := is.store.GetClusterInstallations(
		&model.ClusterInstallationFilter{
			Paging:         model.AllPagesNotDeleted(),
			InstallationID: installation.ID,
		})
	if err != nil {
		return err
	}

	if len(clusterInstallations) < 1 {
		return errors.Errorf("no ClusterInstallations found for Installation %s", installation.ID)
	}
	ci := clusterInstallations[0]

	cluster, err := is.store.GetCluster(ci.ClusterID)
	if err != nil {
		return err
	}

	// ensure that the Installation will be able to accept the import
	mmctl := &mmctl{cluster: cluster, clusterInstallation: ci, ImportSupervisor: is}
	err = is.ensureTeamSettings(mmctl, imprt)
	if err != nil {
		return err
	}

	srcKey, err := is.copyImportToWorkspaceFilestore(imprt, installation)
	if err != nil {
		return errors.Wrapf(err, "failed to copy workspace import archive to Installation %s filestore", installation.ID)
	}

	return is.startImportProcessAndWait(mmctl, srcKey, imprt.ID)
}

func (is *ImportSupervisor) teamAlreadyExists(mmctl *mmctl, teamName string) (bool, error) {
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

func (is *ImportSupervisor) ensureTeamSettings(mmctl *mmctl, imprt *awat.ImportStatus) error {
	// ensure that the team exists
	found, err := is.teamAlreadyExists(mmctl, imprt.Team)
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

func (is *ImportSupervisor) getBucketForInstallation(installation *model.Installation) (string, error) {
	switch installation.Filestore {
	// TODO handle single tenant bucket names
	case model.InstallationFilestoreMultiTenantAwsS3, model.InstallationFilestoreBifrost:
		return is.awsClient.GetMultitenantBucketNameForInstallation(installation.ID, is.store)
	case model.InstallationFilestoreAwsS3, model.InstallationFilestoreMinioOperator:
		return "", errors.Errorf("support for workspace imports to workspaces with the filestore type %s not yet supported", installation.Filestore)
	default:
		return "", errors.Errorf("encountered unknown filestore type %s", installation.Filestore)
	}
}

// copyImportToWorkspaceFilestorecopies the archive from the AWAT's bucket to the Installation's
// bucket. Returns the key of the archive in the source bucket or an
// error
func (is *ImportSupervisor) copyImportToWorkspaceFilestore(imprt *awat.ImportStatus, installation *model.Installation) (string, error) {
	// calculate necessary paths and copy the archive to the Installation's S3 bucket
	source := strings.SplitN(imprt.Resource, "/", 2)
	if len(source) != 2 {
		return "", errors.Errorf("failed to parse bucket/key from Import %s Resource %s", imprt.ID, imprt.Resource)
	}
	srcBucket := source[0]
	srcKey := source[1]
	destKey := fmt.Sprintf("%s/import/%s", installation.ID, srcKey)

	destBucket, err := is.getBucketForInstallation(installation)
	if err != nil {
		return "", errors.Wrapf(err, "failed to determine bucket name for Installation %s", installation.ID)
	}
	is.logger.Debugf("copying %s/%s to %s/%s", srcBucket, srcKey, destBucket, destKey)
	err = is.awsClient.S3LargeCopy(&srcBucket, &srcKey, &destBucket, &destKey)
	if err != nil {
		return "", errors.Wrapf(err, "failed to copy archive to Installation %s", installation.ID)
	}

	return srcKey, nil
}

func (is *ImportSupervisor) startImportProcessAndWait(mmctl *mmctl, importArchiveFilename string, awatImportID string) error {
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
	is.logger.Infof("Started Import job %s", jobID)
	err = is.waitForImportToComplete(mmctl, jobID, awatImportID)
	errorString := ""
	if err != nil {
		errorString = err.Error()
	}

	cerr := is.awatClient.CompleteImport(
		&awat.ImportCompletedWorkRequest{
			ID:         awatImportID,
			CompleteAt: time.Now().UnixNano() / int64(time.Millisecond),
			Error:      errorString,
		})
	if cerr != nil {
		// don't fail the whole process if the AWAT can't be notified properly
		is.logger.WithError(cerr).Errorf("failed to notify AWAT that Import %s is complete", jobID)
	}

	return err
}

func (is *ImportSupervisor) waitForImportToComplete(mmctl *mmctl, mattermostJobID string, awatImportID string) error {
	complete := false
	var (
		resp         *jobResponse
		output       []byte
		jobResponses []*jobResponse
	)
	var err error = nil

	for !complete {
		err = nil
		output, err = mmctl.Run("import", "job", "show", mattermostJobID)
		if err != nil {
			is.logger.WithError(err).Warn("failed to check job")
			err = nil
			time.Sleep(5 * time.Second)
			continue
		}

		err = json.Unmarshal([]byte(output), &jobResponses)
		if err != nil {
			is.logger.WithError(err).Warn("failed to check job; bad JSON")
			err = nil
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
			return errors.Errorf("import job failed with error %s on line %s", resp.Error, resp.LineNumber)
		}

		is.logger.Debugf("Waiting for job %s to complete. Status: %s Progress: %d", resp.ID, resp.Status, resp.Progress)
		time.Sleep(5 * time.Second)
	}

	is.logger.Infof("Import Job %s successfully completed", resp.ID)
	is.logger.Debugf("Completed with output %s", output)
	return nil
}
