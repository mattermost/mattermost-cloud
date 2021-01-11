// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// S3MultitenantFilestore is a filestore backed by a shared AWS S3 bucket.
type S3MultitenantFilestore struct {
	installationID string
	awsClient      *Client
}

// NewS3MultitenantFilestore returns a new NewS3MultitenantFilestore interface.
func NewS3MultitenantFilestore(installationID string, awsClient *Client) *S3MultitenantFilestore {
	return &S3MultitenantFilestore{
		installationID: installationID,
		awsClient:      awsClient,
	}
}

// Provision completes all the steps necessary to provision an S3 multitenant
// filestore.
func (f *S3MultitenantFilestore) Provision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	err := f.s3FilestoreProvision(store, logger)
	if err != nil {
		return errors.Wrap(err, "failed to provision AWS multitenant S3 filestore")
	}

	return nil
}

// Teardown removes all AWS resources related to a shared S3 filestore.
func (f *S3MultitenantFilestore) Teardown(keepData bool, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	awsID := CloudID(f.installationID)

	logger = logger.WithFields(log.Fields{
		"awsID":          awsID,
		"filestore-type": "s3-multitenant",
	})
	logger.Info("Tearing down AWS S3 filestore")

	bucketName, err := getMultitenantBucketNameForInstallation(f.installationID, store, f.awsClient)
	if err != nil {
		// Perform a manual check to see if no cluster installations were ever
		// created for this installation.
		clusterInstallations, ciErr := store.GetClusterInstallations(&model.ClusterInstallationFilter{
			PerPage:        model.AllPerPage,
			InstallationID: f.installationID,
			IncludeDeleted: true,
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

	err = f.awsClient.iamEnsureUserDeleted(awsID, logger)
	if err != nil {
		return errors.Wrap(err, "failed to delete AWS IAM user")
	}

	err = f.awsClient.secretsManagerEnsureIAMAccessKeySecretDeleted(awsID, logger)
	if err != nil {
		return errors.Wrap(err, "failed to delete IAM access key secret")
	}

	if keepData {
		logger.Info("AWS S3 bucket was left intact due to the keep-data setting of this server")
		return nil
	}

	err = f.awsClient.S3EnsureBucketDirectoryDeleted(bucketName, f.installationID, logger)
	if err != nil {
		return errors.Wrap(err, "unable to ensure that AWS S3 filestore was deleted")
	}

	logger.Debug("AWS multitenant S3 filestore was deleted")
	return nil
}

// GenerateFilestoreSpecAndSecret creates the k8s filestore spec and secret for
// accessing the shared S3 bucket.
func (f *S3MultitenantFilestore) GenerateFilestoreSpecAndSecret(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*model.FilestoreConfig, *corev1.Secret, error) {
	awsID := CloudID(f.installationID)

	logger = logger.WithFields(log.Fields{
		"awsID":          awsID,
		"filestore-type": "s3-multitenant",
	})
	logger.Debug("Generating S3 multitenant filestore information")

	bucketName, err := getMultitenantBucketNameForInstallation(f.installationID, store, f.awsClient)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to find multitenant bucket")
	}

	logger = logger.WithField("s3-bucket-name", bucketName)

	iamAccessKey, err := f.awsClient.secretsManagerGetIAMAccessKey(awsID)
	if err != nil {
		return nil, nil, err
	}

	filestoreSecretName := fmt.Sprintf("%s-iam-access-key", f.installationID)
	filestoreSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: filestoreSecretName,
		},
		StringData: map[string]string{
			"accesskey": iamAccessKey.ID,
			"secretkey": iamAccessKey.Secret,
		},
	}

	S3RegionURL := S3URL
	awsRegion := *f.awsClient.config.Region
	if awsRegion != "" && awsRegion != "us-east-1" {
		S3RegionURL = "s3." + awsRegion + ".amazonaws.com"
	}

	filestoreConfig := &model.FilestoreConfig{
		URL:    S3RegionURL,
		Bucket: bucketName,
		Secret: filestoreSecretName,
	}

	logger.Debug("AWS multitenant S3 filestore configuration generated for cluster installation")

	return filestoreConfig, filestoreSecret, nil
}

// s3FilestoreProvision provisions a shared S3 filestore for an installation.
func (f *S3MultitenantFilestore) s3FilestoreProvision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	awsID := CloudID(f.installationID)

	logger = logger.WithFields(log.Fields{
		"awsID":          awsID,
		"filestore-type": "s3-multitenant",
	})
	logger.Info("Provisioning AWS multitenant S3 filestore")

	bucketName, err := getMultitenantBucketNameForInstallation(f.installationID, store, f.awsClient)
	if err != nil {
		return errors.Wrap(err, "failed to find multitenant bucket")
	}

	logger = logger.WithField("s3-bucket-name", bucketName)

	user, err := f.awsClient.iamEnsureUserCreated(awsID, logger)
	if err != nil {
		return err
	}

	// The IAM policy lookup requires the AWS account ID for the ARN. The user
	// object contains this ID so we will user that.
	arn, err := arn.Parse(*user.Arn)
	if err != nil {
		return err
	}
	policyARN := fmt.Sprintf("arn:aws:iam::%s:policy/%s", arn.AccountID, awsID)
	policy, err := f.awsClient.iamEnsureS3PolicyCreated(awsID, policyARN, bucketName, f.installationID, logger)
	if err != nil {
		return err
	}
	err = f.awsClient.iamEnsurePolicyAttached(awsID, policyARN, logger)
	if err != nil {
		return err
	}
	logger.WithFields(log.Fields{
		"iam-policy-name": *policy.PolicyName,
		"iam-user-name":   *user.UserName,
	}).Debug("AWS IAM policy attached to user")

	ak, err := f.awsClient.iamEnsureAccessKeyCreated(awsID, logger)
	if err != nil {
		return err
	}
	logger.WithField("iam-user-name", *user.UserName).Debug("AWS IAM user access key created")

	err = f.awsClient.secretsManagerEnsureIAMAccessKeySecretCreated(awsID, ak, logger)
	if err != nil {
		return err
	}
	logger.WithField("iam-user-name", *user.UserName).Debug("AWS secrets manager secret created")

	return nil
}
