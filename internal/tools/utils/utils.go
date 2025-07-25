// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utils

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// CopyDirectory copy the entire directory to another destination
func CopyDirectory(source string, dest string) error {
	sourceinfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	err = os.MkdirAll(dest, sourceinfo.Mode())
	if err != nil {
		return err
	}

	directory, _ := os.Open(source)
	objects, err := directory.Readdir(-1)
	if err != nil {
		return errors.Wrap(err, "error reading source directory")
	}

	for _, obj := range objects {
		sourcefilepointer := source + "/" + obj.Name()
		destinationfilepointer := dest + "/" + obj.Name()

		if obj.IsDir() {
			err = CopyDirectory(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println(err)
			}
		} else {
			err = copyFile(sourcefilepointer, destinationfilepointer)
			if err != nil {
				fmt.Println(err)
			}
		}

	}
	return nil
}

func copyFile(source string, dest string) error {
	sourcefile, err := os.Open(source)
	if err != nil {
		return err
	}

	defer sourcefile.Close()

	destfile, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer destfile.Close()

	_, err = io.Copy(destfile, sourcefile)
	if err == nil {
		sourceinfo, err := os.Stat(source)
		if err != nil {
			_ = os.Chmod(dest, sourceinfo.Mode())
		}
	}

	return nil
}

// ResourceUtil is used for calling any filestore type.
type ResourceUtil struct {
	awsClient                    *aws.Client
	instanceID                   string
	dbClusterUtilizationSettings DBClusterUtilizationSettings
	disableDBCheck               bool
	enableS3Versioning           bool
}

// DBClusterUtilizationSettings define maximum utilization of database clusters.
type DBClusterUtilizationSettings struct {
	MaxInstallationsPerseus              int
	MaxInstallationsRDSPostgresPGBouncer int
	MaxInstallationsRDSPostgres          int
	MaxInstallationsRDSMySQL             int
}

// NewResourceUtil returns a new instance of ResourceUtil.
func NewResourceUtil(
	instanceID string,
	awsClient *aws.Client,
	dbClusterUtilization DBClusterUtilizationSettings,
	disableDBCheck bool,
	enableS3Versioning bool) *ResourceUtil {
	return &ResourceUtil{
		awsClient:                    awsClient,
		instanceID:                   instanceID,
		dbClusterUtilizationSettings: dbClusterUtilization,
		disableDBCheck:               disableDBCheck,
		enableS3Versioning:           enableS3Versioning,
	}
}

// GetFilestore returns the Filestore interface that matches the installation.
func (r *ResourceUtil) GetFilestore(installation *model.Installation) model.Filestore {
	switch installation.Filestore {
	case model.InstallationFilestoreMinioOperator:
		return model.NewMinioOperatorFilestore()
	case model.InstallationFilestoreLocalEphemeral:
		return model.NewLocalEphemeralFilestore()
	case model.InstallationFilestoreAwsS3:
		return aws.NewS3Filestore(installation.ID, r.awsClient, r.enableS3Versioning)
	case model.InstallationFilestoreMultiTenantAwsS3:
		return aws.NewS3MultitenantFilestore(installation.ID, r.awsClient)
	case model.InstallationFilestoreBifrost:
		return aws.NewBifrostFilestore(installation.ID, r.awsClient)
	}

	// Warning: we should never get here as it would mean that we didn't match
	// our filestore type.
	return model.NewMinioOperatorFilestore()
}

// GetDatabaseForInstallation returns the Database interface that matches the installation.
func (r *ResourceUtil) GetDatabaseForInstallation(installation *model.Installation) model.Database {
	return r.GetDatabase(installation.ID, installation.Database)
}

// GetDatabase returns the Database interface that matches the installationID and DB type.
func (r *ResourceUtil) GetDatabase(installationID, dbType string) model.Database {
	switch dbType {
	case model.InstallationDatabaseMysqlOperator:
		return model.NewMysqlOperatorDatabase()
	case model.InstallationDatabaseExternal:
		return aws.NewExternalDatabase(installationID, r.awsClient)
	case model.InstallationDatabaseSingleTenantRDSMySQL:
		return aws.NewRDSDatabase(model.DatabaseEngineTypeMySQL, installationID, r.awsClient, r.disableDBCheck)
	case model.InstallationDatabaseSingleTenantRDSPostgres:
		return aws.NewRDSDatabase(model.DatabaseEngineTypePostgres, installationID, r.awsClient, r.disableDBCheck)
	case model.InstallationDatabaseMultiTenantRDSMySQL:
		return aws.NewRDSMultitenantDatabase(
			model.DatabaseEngineTypeMySQL,
			r.instanceID,
			installationID,
			r.awsClient,
			r.dbClusterUtilizationSettings.MaxInstallationsRDSMySQL,
			r.disableDBCheck,
		)
	case model.InstallationDatabaseMultiTenantRDSPostgres:
		return aws.NewRDSMultitenantDatabase(
			model.DatabaseEngineTypePostgres,
			r.instanceID,
			installationID,
			r.awsClient,
			r.dbClusterUtilizationSettings.MaxInstallationsRDSPostgres,
			r.disableDBCheck,
		)
	case model.InstallationDatabaseMultiTenantRDSPostgresPGBouncer:
		return aws.NewRDSMultitenantPGBouncerDatabase(
			model.DatabaseEngineTypePostgresProxy,
			r.instanceID,
			installationID,
			r.awsClient,
			r.dbClusterUtilizationSettings.MaxInstallationsRDSPostgresPGBouncer,
			r.disableDBCheck,
		)
	case model.InstallationDatabasePerseus:
		return aws.NewPerseusDatabase(
			model.DatabaseEngineTypePostgresProxyPerseus,
			r.instanceID,
			installationID,
			r.awsClient,
			r.dbClusterUtilizationSettings.MaxInstallationsPerseus,
			r.disableDBCheck,
		)
	}

	// Warning: we should never get here as it would mean that we didn't match
	// our database type.
	return model.NewMysqlOperatorDatabase()
}

// EnsureSecretManagerSecretDeleted ensures a secret with the provided name is
// marked for deletion in AWS Secrets Manager.
func (r *ResourceUtil) EnsureSecretManagerSecretDeleted(secretName string, logger log.FieldLogger) error {
	return r.awsClient.SecretsManagerEnsureSecretDeleted(secretName, logger)
}

// Retry is retrying a function for a maximum number of attempts and time
func Retry(attempts int, sleep time.Duration, f func() error) error {
	if err := f(); err != nil {
		if attempts--; attempts > 0 {
			// Add some randomness to prevent creating a Thundering Herd
			jitter := time.Duration(rand.Int63n(int64(sleep)))
			sleep = sleep + jitter/2

			time.Sleep(sleep)
			return Retry(attempts, 2*sleep, f)
		}
		return err
	}

	return nil
}
