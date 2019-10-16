package model

import (
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	log "github.com/sirupsen/logrus"

	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
	corev1 "k8s.io/api/core/v1"
)

const (
	// InstallationFilestoreMinioOperator is a filestore hosted in kubernetes via the operator.
	InstallationFilestoreMinioOperator = "minio-operator"
	// InstallationFilestoreAwsS3 is a filestore hosted via Amazon S3.
	InstallationFilestoreAwsS3 = "aws-s3"
)

// Filestore is the interface for managing Mattermost filestores.
type Filestore interface {
	Provision(logger log.FieldLogger) error
	Teardown(keepData bool, logger log.FieldLogger) error
	GenerateFilestoreSpecAndSecret(logger log.FieldLogger) (*mmv1alpha1.Minio, *corev1.Secret, error)
}

// MinioOperatorFilestore is a filestore backed by the MinIO operator.
type MinioOperatorFilestore struct{}

// NewMinioOperatorFilestore returns a new NewMinioOperatorFilestore interface.
func NewMinioOperatorFilestore() *MinioOperatorFilestore {
	return &MinioOperatorFilestore{}
}

// Provision completes all the steps necessary to provision a MinIO operator filestore.
func (f *MinioOperatorFilestore) Provision(logger log.FieldLogger) error {
	logger.Info("MinIO operator filestore requires no pre-provisioning; skipping...")

	return nil
}

// Teardown removes all MinIO operator resources related to an S3 filestore.
func (f *MinioOperatorFilestore) Teardown(keepData bool, logger log.FieldLogger) error {
	logger.Info("MinIO operator filestore requires no teardown; skipping...")
	if keepData {
		logger.Warn("Data preservation was requested, but isn't currently possible with the MinIO operator")
	}

	return nil
}

// GenerateFilestoreSpecAndSecret creates the k8s filestore spec and secret for
// accessing the MinIO operator filestore.
func (f *MinioOperatorFilestore) GenerateFilestoreSpecAndSecret(logger log.FieldLogger) (*mmv1alpha1.Minio, *corev1.Secret, error) {
	return nil, nil, nil
}

// GetFilestore returns the Filestore interface that matches the installation.
func (i *Installation) GetFilestore() Filestore {
	switch i.Filestore {
	case InstallationFilestoreMinioOperator:
		return NewMinioOperatorFilestore()
	case InstallationFilestoreAwsS3:
		return aws.NewS3Filestore(i.ID)
	}

	// Warning: we should never get here as it would mean that we didn't match
	// our filestore type.
	return NewMinioOperatorFilestore()
}

// InternalFilestore returns true if the installation's filestore is internal
// to the kubernetes cluster it is running on.
func (i *Installation) InternalFilestore() bool {
	return i.Filestore == InstallationFilestoreMinioOperator
}

// IsSupportedFilestore returns true if the given filestore string is supported.
func IsSupportedFilestore(filestore string) bool {
	return filestore == InstallationFilestoreMinioOperator || filestore == InstallationFilestoreAwsS3
}
