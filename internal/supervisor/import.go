// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"fmt"
	"strings"

	awat "github.com/mattermost/awat/model"
	toolsAWS "github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

type ImportSupervisor struct {
	awsClient  *toolsAWS.Client
	awatClient *awat.Client
	logger     logrus.FieldLogger
	store      installationStore
	ID         string
}

func NewImportSupervisor(awsClient *toolsAWS.Client, awat *awat.Client, store installationStore, logger logrus.FieldLogger) *ImportSupervisor {
	return &ImportSupervisor{
		awsClient:  awsClient,
		awatClient: awat,
		store:      store,
		logger:     logger,

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
		return errors.Wrap(err, "failed to get a ready Translation from the AWAT")
	}
	if work == nil {
		// nothing to do
		return nil
	}

	err = is.importTranslation(work)
	if err != nil {
		is.logger.WithError(err).Error("failed to translate")
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

	destKey := fmt.Sprintf("%s/import/%s", installation.ID, imprt.Resource)
	source := strings.SplitN(imprt.Resource, "/", 2)
	if len(source) != 2 {
		return errors.Errorf("failed to parse bucket/key from Import %s Resource %s", imprt.ID, imprt.Resource)
	}
	srcBucket := source[0]
	srcKey := source[1]

	// XXX TODO handle single tenant bucket names
	destBucket, err := toolsAWS.GetMultitenantBucketNameForInstallation(installation.ID, is.store, is.awsClient)

	is.logger.Debugf("copying %s/%s to %s/%s", srcBucket, srcKey, destBucket, destKey)
	is.awsClient.S3LargeCopy(&srcBucket, &srcKey, &destBucket, &destKey)
	return nil
}
