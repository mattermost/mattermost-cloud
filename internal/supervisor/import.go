// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	awat "github.com/mattermost/awat/model"
	"github.com/mattermost/mattermost-cloud/internal/provisioner"
	toolsAWS "github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const (
	TEAM_NAME_MAX_LENGTH = 64
	TEAM_NAME_MIN_LENGTH = 2
)

type ImportSupervisor struct {
	awsClient   *toolsAWS.Client
	awatClient  *awat.Client
	logger      logrus.FieldLogger
	store       installationStore
	provisioner *provisioner.KopsProvisioner
	ID          string
}

type mmctl struct {
	cluster             *model.Cluster
	clusterInstallation *model.ClusterInstallation
	installation        *model.Installation

	*ImportSupervisor
}

func (m *mmctl) Do(args ...string) ([]byte, error) {
	args = append([]string{"mmctl", "--format", "json", "--local"}, args...)
	return m.provisioner.ExecClusterInstallationCLI(m.cluster, m.clusterInstallation, args...)
}

type team struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func NewImportSupervisor(awsClient *toolsAWS.Client, awat *awat.Client, store installationStore, provisioner *provisioner.KopsProvisioner, logger logrus.FieldLogger) *ImportSupervisor {
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

	err = is.importTranslation(work)
	if err != nil {
		is.logger.WithError(err).Error("failed to import")
	}
	return err
}

func (is *ImportSupervisor) Shutdown() {
	is.logger.Debug("Shutting down import supervisor")
}

func (is *ImportSupervisor) importTranslation(imprt *awat.ImportStatus) error {
	// get installation metadata
	installation, err := is.store.GetInstallation(imprt.InstallationID, false, false)
	if err != nil {
		return err
	}
	if installation == nil {
		return errors.Errorf("Installation %s not found for Import %s", imprt.InstallationID, imprt.ID)
	}

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

	return is.startImportProcessAndWait(mmctl, srcKey)
}

func (is *ImportSupervisor) ensureTeamSettings(mmctl *mmctl, imprt *awat.ImportStatus) error {
	output, err := mmctl.Do("team", "search", imprt.Team)
	if err != nil {
		return err
	}
	teamSearch := []*team{}
	_ = json.Unmarshal(output, &teamSearch)

	found := false
	for _, team := range teamSearch {
		if team.Name == imprt.Team {
			found = true
			break
		}
	}

	if !found {
		is.logger.Debugf("failed to find team with name \"%s\"; creating..", imprt.Team)
		teamName := cleanTeamName(imprt.Team)
		output, err = mmctl.Do("team", "create", "--name", teamName, "--display_name", teamName)
		if err != nil {
			return errors.Wrapf(err, "failed to find or create team %s; full output was:%s\n", teamName, string(output))
		}
	}

	output, err = mmctl.Do("config", "get", "TeamSettings.MaxUsersPerTeam")
	if err != nil {
		return err
	}

	// despite requesting JSON mmctl always just returns a bare number
	// for these config commands
	currentMaxUsers, err := strconv.Atoi(strings.TrimSpace(string(output)))
	if err != nil {
		return errors.Wrapf(err, "failed to convert \"%s\" to a number", output)
	}

	// ensure that there will be enough new user slots for this import
	maxUsers := currentMaxUsers + imprt.Users
	_, err = mmctl.Do("config", "set", "TeamSettings.MaxUsersPerTeam", strconv.Itoa(maxUsers))
	if err != nil {
		return errors.Wrapf(err, "failed to add %d users to MaxUsersPerTeam", imprt.Users)
	}

	return nil
}

func (is *ImportSupervisor) getBucketForInstallation(installation *model.Installation) (string, error) {
	switch installation.Filestore {
	// TODO handle single tenant bucket names
	case model.InstallationFilestoreMultiTenantAwsS3:
		fallthrough
	case model.InstallationFilestoreBifrost:
		return toolsAWS.GetMultitenantBucketNameForInstallation(installation.ID, is.store, is.awsClient)
	case model.InstallationFilestoreAwsS3:
		fallthrough
	case model.InstallationFilestoreMinioOperator:
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

func (is *ImportSupervisor) startImportProcessAndWait(mmctl *mmctl, srcKey string) error {
	output, err := mmctl.Do("import", "process", srcKey)
	if err != nil {
		return err
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

	complete := false
	for !complete {
		checkResponses := []*jobResponse{}
		output, err = mmctl.Do("import", "job", "show", jobID)

		err = json.Unmarshal([]byte(output), &checkResponses)
		if err != nil {
			is.logger.WithError(err).Warn("failed to check job")
			continue
		}
		if len(checkResponses) != 1 {
			return errors.Errorf("unexpected number of responses from jobs API endpoint (%d)", len(checkResponses))
		}
		resp := checkResponses[0]
		if resp.Status != "pending" && resp.Status != "in_progress" {
			complete = true
		}
		if resp.Status == "error" {
			err = errors.New("import job failed")
		}

		is.logger.Debugf("Waiting for job to complete; response was %+v", resp)
		time.Sleep(5 * time.Second)
	}

	is.logger.Infof("Completed with output %s", output)

	return nil
}

type jobResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

var reservedTeamNames = []string{
	"admin",
	"api",
	"channel",
	"claim",
	"error",
	"files",
	"help",
	"landing",
	"login",
	"mfa",
	"oauth",
	"plug",
	"plugins",
	"post",
	"signup",
}

var validTeamNameCharacter = regexp.MustCompile(`^[a-z0-9-]$`)

// lifted from mattermost-server
func cleanTeamName(s string) string {
	s = strings.ToLower(strings.Replace(s, " ", "-", -1))

	for _, value := range reservedTeamNames {
		if strings.Index(s, value) == 0 {
			s = strings.Replace(s, value, "", -1)
		}
	}

	s = strings.TrimSpace(s)

	for _, c := range s {
		char := fmt.Sprintf("%c", c)
		if !validTeamNameCharacter.MatchString(char) {
			s = strings.Replace(s, char, "", -1)
		}
	}

	s = strings.Trim(s, "-")

	if !isValidTeamName(s) {
		s = model.NewID()
	}

	return s
}

// Also lifted from mattermost-server but with the addition of a check
// for max length
func isValidTeamName(s string) bool {
	if !isValidAlphaNum(s) {
		return false
	}

	if len(s) < TEAM_NAME_MIN_LENGTH ||
		len(s) > TEAM_NAME_MAX_LENGTH {
		return false
	}

	return true
}

func isValidAlphaNum(s string) bool {
	validAlphaNum := regexp.MustCompile(`^[a-z0-9]+([a-z\-0-9]+|(__)?)[a-z0-9]+$`)

	return validAlphaNum.MatchString(s)
}
