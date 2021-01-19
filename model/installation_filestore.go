// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	log "github.com/sirupsen/logrus"

	corev1 "k8s.io/api/core/v1"
)

const (
	// InstallationFilestoreMinioOperator is a filestore hosted in kubernetes
	// via the operator.
	InstallationFilestoreMinioOperator = "minio-operator"
	// InstallationFilestoreAwsS3 is a filestore hosted via Amazon S3.
	InstallationFilestoreAwsS3 = "aws-s3"
	// InstallationFilestoreMultiTenantAwsS3 is a filestore hosted via a shared
	// Amazon S3 bucket.
	InstallationFilestoreMultiTenantAwsS3 = "aws-multitenant-s3"
	// InstallationFilestoreBifrost is a filestore hosted via a shared Amazon S3
	// bucket using the bifrost gateway.
	InstallationFilestoreBifrost = "bifrost"
)

// Filestore is the interface for managing Mattermost filestores.
type Filestore interface {
	Provision(store InstallationDatabaseStoreInterface, logger log.FieldLogger) error
	Teardown(keepData bool, store InstallationDatabaseStoreInterface, logger log.FieldLogger) error
	GenerateFilestoreSpecAndSecret(store InstallationDatabaseStoreInterface, logger log.FieldLogger) (*FilestoreConfig, *corev1.Secret, error)
}

// FilestoreConfig represent universal configuration of the File store.
type FilestoreConfig struct {
	URL    string
	Bucket string
	Secret string
}

// MinioOperatorFilestore is a filestore backed by the MinIO operator.
type MinioOperatorFilestore struct{}

// NewMinioOperatorFilestore returns a new NewMinioOperatorFilestore interface.
func NewMinioOperatorFilestore() *MinioOperatorFilestore {
	return &MinioOperatorFilestore{}
}

// Provision completes all the steps necessary to provision a MinIO operator filestore.
func (f *MinioOperatorFilestore) Provision(store InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	logger.Info("MinIO operator filestore requires no pre-provisioning; skipping...")

	return nil
}

// Teardown removes all MinIO operator resources related to a given installation.
func (f *MinioOperatorFilestore) Teardown(keepData bool, store InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	logger.Info("MinIO operator filestore requires no teardown; skipping...")
	if keepData {
		logger.Warn("Data preservation was requested, but isn't currently possible with the MinIO operator")
	}

	return nil
}

// GenerateFilestoreSpecAndSecret creates the k8s filestore spec and secret for
// accessing the MinIO operator filestore.
func (f *MinioOperatorFilestore) GenerateFilestoreSpecAndSecret(store InstallationDatabaseStoreInterface, logger log.FieldLogger) (*FilestoreConfig, *corev1.Secret, error) {
	return nil, nil, nil
}

// InternalFilestore returns true if the installation's filestore is internal
// to the kubernetes cluster it is running on.
func (i *Installation) InternalFilestore() bool {
	return i.Filestore == InstallationFilestoreMinioOperator
}

// IsSupportedFilestore returns true if the given filestore string is supported.
func IsSupportedFilestore(filestore string) bool {
	return filestore == InstallationFilestoreMinioOperator ||
		filestore == InstallationFilestoreAwsS3 ||
		filestore == InstallationFilestoreMultiTenantAwsS3 ||
		filestore == InstallationFilestoreBifrost
}
