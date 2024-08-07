// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws/arn"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// S3Filestore is a filestore backed by AWS S3.
type S3Filestore struct {
	installationID   string
	awsClient        *Client
	enableVersioning bool
}

// NewS3Filestore returns a new S3Filestore interface.
func NewS3Filestore(installationID string, awsClient *Client, enableVersioning bool) *S3Filestore {
	return &S3Filestore{
		installationID:   installationID,
		awsClient:        awsClient,
		enableVersioning: enableVersioning,
	}
}

// Provision completes all the steps necessary to provision an S3 filestore.
func (f *S3Filestore) Provision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	err := f.s3FilestoreProvision(f.installationID, logger)
	if err != nil {
		return errors.Wrap(err, "unable to provision AWS S3 filestore")
	}

	return nil
}

// Teardown removes all AWS resources related to an S3 filestore.
func (f *S3Filestore) Teardown(keepData bool, store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	awsID := CloudID(f.installationID)

	logger = logger.WithField("s3-bucket-name", awsID)
	logger.Info("Tearing down AWS S3 filestore")

	err := f.awsClient.iamEnsureUserDeleted(awsID, logger)
	if err != nil {
		return errors.Wrap(err, "unable to teardown AWS S3 filestore")
	}

	err = f.awsClient.secretsManagerEnsureIAMAccessKeySecretDeleted(awsID, logger)
	if err != nil {
		return errors.Wrap(err, "unable to teardown AWS S3 filestore")
	}

	if keepData {
		logger.Info("AWS S3 bucket was left intact due to the keep-data setting of this server")
		return nil
	}

	err = f.awsClient.S3EnsureBucketDeleted(awsID, logger)
	if err != nil {
		return errors.Wrap(err, "unable to ensure that AWS S3 filestore was deleted")
	}

	logger.Debug("AWS S3 bucket was deleted")
	return nil
}

// GenerateFilestoreSpecAndSecret creates the k8s filestore spec and secret for
// accessing the S3 bucket.
func (f *S3Filestore) GenerateFilestoreSpecAndSecret(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*model.FilestoreConfig, *corev1.Secret, error) {
	awsID := CloudID(f.installationID)
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

	S3RegionURL := f.awsClient.GetS3RegionURL()

	filestoreConfig := &model.FilestoreConfig{
		URL:    S3RegionURL,
		Bucket: awsID,
		Secret: filestoreSecretName,
	}

	logger.Debug("Cluster installation configured to use an AWS S3 filestore")

	return filestoreConfig, filestoreSecret, nil
}

// s3FilestoreProvision provisions an S3 filestore for an installation.
func (f *S3Filestore) s3FilestoreProvision(installationID string, logger log.FieldLogger) error {
	logger.Info("Provisioning AWS S3 filestore")

	awsID := CloudID(installationID)

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
	policy, err := f.awsClient.iamEnsureS3PolicyCreated(awsID, policyARN, awsID, "*", logger)
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

	err = f.awsClient.s3EnsureBucketCreated(awsID, f.enableVersioning)
	if err != nil {
		return err
	}
	logger.WithField("s3-bucket-name", awsID).Debug("AWS S3 bucket created")

	ak, err := f.awsClient.iamEnsureAccessKeyCreated(awsID, logger)
	if err != nil {
		return err
	}
	logger.WithField("iam-user-name", *user.UserName).Debug("AWS IAM user access key created")

	err = f.awsClient.secretsManagerEnsureIAMAccessKeySecretCreated(awsID, ak)
	if err != nil {
		return err
	}
	logger.WithField("iam-user-name", *user.UserName).Debug("AWS secrets manager secret created")

	return nil
}
