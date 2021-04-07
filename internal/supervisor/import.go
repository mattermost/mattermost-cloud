// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	awat "github.com/mattermost/awat/model"
	"github.com/mattermost/mattermost-cloud/internal/provisioner"
	toolsAWS "github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type ImportSupervisor struct {
	awsClient   *toolsAWS.Client
	awatClient  *awat.Client
	logger      logrus.FieldLogger
	store       installationStore
	provisioner *provisioner.KopsProvisioner
	ID          string
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

func (is ImportSupervisor) importTranslation(imprt *awat.ImportStatus) error {
	installation, err := is.store.GetInstallation(imprt.InstallationID, false, false)
	if err != nil {
		return err
	}
	if installation == nil {
		return errors.Errorf("Installation %s not found for Import %s", imprt.InstallationID, imprt.ID)
	}

	source := strings.SplitN(imprt.Resource, "/", 2)
	if len(source) != 2 {
		return errors.Errorf("failed to parse bucket/key from Import %s Resource %s", imprt.ID, imprt.Resource)
	}
	srcBucket := source[0]
	srcKey := source[1]
	destKey := fmt.Sprintf("%s/import/%s", installation.ID, srcKey)

	// XXX TODO handle single tenant bucket names
	destBucket, err := toolsAWS.GetMultitenantBucketNameForInstallation(installation.ID, is.store, is.awsClient)

	is.logger.Debugf("copying %s/%s to %s/%s", srcBucket, srcKey, destBucket, destKey)
	err = is.awsClient.S3LargeCopy(&srcBucket, &srcKey, &destBucket, &destKey)
	if err != nil {
		return err
	}

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

	output, err := is.provisioner.ExecClusterInstallationCLI(cluster, ci,
		"mmctl", "import", "process", "--local", srcKey, "--format", "json",
	)
	if err != nil {
		return err
	}

	jobResponses := []*jobResponse{}
	err = json.Unmarshal([]byte(output), &jobResponses)
	if err != nil {
		return err
	}

	if len(jobResponses) < 1 {
		return errors.New("response was empty")
	}

	is.logger.Infof("Started Import job %+v", jobResponses[0])
	jobID := jobResponses[0].ID

	complete := false
	for !complete {
		checkResponses := []*jobResponse{}
		output, err = is.provisioner.ExecClusterInstallationCLI(cluster, ci,
			"mmctl", "import", "job", "--local", "show", jobID, "--format", "json")

		err = json.Unmarshal([]byte(output), &checkResponses)
		if err != nil {
			is.logger.WithError(err).Warn("failed to check job")
			continue
		}
		if len(checkResponses) != 1 {
			return errors.Errorf("XXX %+v", checkResponses)
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
	return err
}

type jobResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}
