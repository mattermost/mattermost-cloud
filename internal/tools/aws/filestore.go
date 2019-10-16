package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// S3Filestore is a filestore backed by AWS S3.
type S3Filestore struct {
	installationID string
}

// NewS3Filestore returns a new S3Filestore interface.
func NewS3Filestore(installationID string) *S3Filestore {
	return &S3Filestore{
		installationID: installationID,
	}
}

// Provision completes all the steps necessary to provision an S3 filestore.
func (f *S3Filestore) Provision(logger log.FieldLogger) error {
	err := s3FilestoreProvision(f.installationID, logger)
	if err != nil {
		return errors.Wrap(err, "unable to provision AWS S3 filestore")
	}

	return nil
}

// Teardown removes all AWS resources related to an S3 filestore.
func (f *S3Filestore) Teardown(keepData bool, logger log.FieldLogger) error {
	err := s3FilestoreTeardown(f.installationID, keepData, logger)
	if err != nil {
		return errors.Wrap(err, "unable to teardown AWS S3 filestore")
	}

	return nil
}

// GenerateFilestoreSpecAndSecret creates the k8s filestore spec and secret for
// accessing the S3 bucket.
func (f *S3Filestore) GenerateFilestoreSpecAndSecret(logger log.FieldLogger) (*mmv1alpha1.Minio, *corev1.Secret, error) {
	awsID := CloudID(f.installationID)
	iamAccessKey, err := secretsManagerGetIAMAccessKey(awsID)
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

	filestoreSpec := &mmv1alpha1.Minio{
		ExternalURL:    S3URL,
		ExternalBucket: awsID,
		Secret:         filestoreSecretName,
	}

	logger.Debug("Cluster installation configured to use an AWS S3 filestore")

	return filestoreSpec, filestoreSecret, nil
}

func s3FilestoreProvision(installationID string, logger log.FieldLogger) error {
	logger.Info("Provisioning AWS S3 filestore")

	a := New("n/a")
	awsID := CloudID(installationID)

	user, err := a.iamEnsureUserCreated(awsID, logger)
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
	policy, err := a.iamEnsurePolicyCreated(awsID, policyARN, logger)
	if err != nil {
		return err
	}
	err = a.iamEnsurePolicyAttached(awsID, policyARN)
	if err != nil {
		return err
	}
	logger.WithFields(log.Fields{
		"iam-policy-name": *policy.PolicyName,
		"iam-user-name":   *user.UserName,
	}).Info("AWS IAM policy attached to user")

	err = a.s3EnsureBucketCreated(awsID)
	if err != nil {
		return err
	}
	logger.WithField("s3-bucket-name", awsID).Info("AWS S3 bucket created")

	ak, err := a.iamEnsureAccessKeyCreated(awsID, logger)
	if err != nil {
		return err
	}
	logger.WithField("iam-user-name", *user.UserName).Info("AWS IAM user access key created")

	err = a.secretsManagerEnsureIAMAccessKeySecretCreated(awsID, ak)
	if err != nil {
		return err
	}
	logger.WithField("iam-user-name", *user.UserName).Info("AWS secrets manager secret created")

	return nil
}

func s3FilestoreTeardown(installationID string, keepBucket bool, logger log.FieldLogger) error {
	a := New("n/a")
	awsID := CloudID(installationID)

	err := a.iamEnsureUserDeleted(awsID, logger)
	if err != nil {
		return err
	}
	err = a.secretsManagerEnsureIAMAccessKeySecretDeleted(awsID, logger)
	if err != nil {
		return err
	}
	logger.WithField("iam-user-name", awsID).Info("AWS secrets manager secret deleted")

	if !keepBucket {
		err = a.s3EnsureBucketDeleted(awsID)
		if err != nil {
			return err
		}
		logger.WithField("s3-bucket-name", awsID).Info("AWS S3 bucket deleted")
	}

	return nil
}
