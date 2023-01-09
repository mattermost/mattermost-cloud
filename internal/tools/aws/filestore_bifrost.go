// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"fmt"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BifrostFilestore is a filestore backed by a shared AWS S3 bucket with access
// controlled by bifrost.
type BifrostFilestore struct {
	installationID string
	awsClient      *Client
}

// NewBifrostFilestore returns a new NewBifrostFilestore interface.
func NewBifrostFilestore(installationID string, awsClient *Client) *BifrostFilestore {
	return &BifrostFilestore{
		installationID: installationID,
		awsClient:      awsClient,
	}
}

// Provision completes all the steps necessary to provision an S3 multitenant
// filestore.
func (f *BifrostFilestore) Provision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	err := f.s3FilestoreProvision(store, logger)
	if err != nil {
		return errors.Wrap(err, "failed to provision bifrost filestore")
	}

	return nil
}

// Teardown removes all AWS resources related to a shared S3 filestore.
func (f *BifrostFilestore) Teardown(keepData bool, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	awsID := CloudID(f.installationID)

	logger = logger.WithFields(log.Fields{
		"awsID":          awsID,
		"filestore-type": "bifrost",
	})
	logger.Info("Tearing down bifrost filestore")

	bucketName, err := f.awsClient.GetMultitenantBucketNameForInstallation(f.installationID, store)
	if err != nil {
		// Perform a manual check to see if no cluster installations were ever
		// created for this installation.
		clusterInstallations, ciErr := store.GetClusterInstallations(&model.ClusterInstallationFilter{
			Paging:         model.AllPagesWithDeleted(),
			InstallationID: f.installationID,
		})
		if ciErr != nil {
			return errors.Wrap(ciErr, "failed to query cluster installations")
		}
		if len(clusterInstallations) == 0 {
			logger.Warn("No cluster installations found for installation; assuming multitenant filestore was never created")
			return nil
		}

		return errors.Wrap(err, "failed to find multitenant bucket")
	}

	logger = logger.WithField("s3-bucket-name", bucketName)

	if keepData {
		logger.Info("AWS S3 bucket was left intact due to the keep-data setting of this server")
		return nil
	}

	err = f.awsClient.S3EnsureBucketDirectoryDeleted(bucketName, f.installationID, logger)
	if err != nil {
		return errors.Wrap(err, "failed to ensure that bifrost filestore was deleted")
	}

	logger.Debug("Bifrost filestore teardown complete")
	return nil
}

// GenerateFilestoreSpecAndSecret creates the k8s filestore spec and secret for
// accessing the shared S3 bucket.
func (f *BifrostFilestore) GenerateFilestoreSpecAndSecret(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*model.FilestoreConfig, *corev1.Secret, error) {
	awsID := CloudID(f.installationID)

	logger = logger.WithFields(log.Fields{
		"awsID":          awsID,
		"filestore-type": "bifrost",
	})
	logger.Debug("Generating Bifrost filestore information")

	bucketName, err := f.awsClient.GetMultitenantBucketNameForInstallation(f.installationID, store)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to find multitenant bucket")
	}

	logger = logger.WithField("s3-bucket-name", bucketName)

	// Although no secrets or credentials are needed for bifrost, the operator
	// expects certain values so we set dummy values instead.
	filestoreSecretName := fmt.Sprintf("%s-iam-access-key", f.installationID)
	filestoreSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: filestoreSecretName,
		},
		StringData: map[string]string{
			"accesskey": "bifrost",
			"secretkey": "bifrost",
		},
	}

	S3RegionURL := f.awsClient.GetS3RegionURL()

	filestoreConfig := &model.FilestoreConfig{
		URL:    S3RegionURL,
		Bucket: bucketName,
		Secret: filestoreSecretName,
	}

	logger.Debug("Bifrost filestore configuration generated for cluster installation")

	return filestoreConfig, filestoreSecret, nil
}

// s3FilestoreProvision provisions a shared S3 filestore for an installation.
func (f *BifrostFilestore) s3FilestoreProvision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	awsID := CloudID(f.installationID)

	logger = logger.WithFields(log.Fields{
		"awsID":          awsID,
		"filestore-type": "bifrost",
	})
	logger.Info("Provisioning bifrost filestore")

	// Bifrost filestores don't require any setup. The only check that we will
	// make is that the multitenant bucket exsists for the shared filestore.
	_, err := f.awsClient.GetMultitenantBucketNameForInstallation(f.installationID, store)
	if err != nil {
		return errors.Wrap(err, "failed to find multitenant bucket")
	}

	return nil
}

// GenerateBifrostUtilitySecret creates the secret needed by the bifrost service
// to access the shared S3 bucket for a given cluster.
func (a *Client) GenerateBifrostUtilitySecret(clusterID string, logger log.FieldLogger) (*corev1.Secret, error) {
	bucketName, err := getMultitenantBucketNameForCluster(clusterID, a)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get multitenant bucket name")
	}

	bifrostIAMCreds, err := a.secretsManagerGetIAMAccessKeyFromSecretName(bucketName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get IAM user credential secret for bifrost")
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "bifrost",
		},
		StringData: map[string]string{
			"Bucket":          bucketName,
			"AccessKeyID":     bifrostIAMCreds.ID,
			"SecretAccessKey": bifrostIAMCreds.Secret,
		},
	}, nil
}
