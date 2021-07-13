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
)

type stop struct {
	error
}

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
			err = os.Chmod(dest, sourceinfo.Mode())
		}

	}

	return nil
}

// ResourceUtil is used for calling any filestore type.
type ResourceUtil struct {
	awsClient  *aws.Client
	instanceID string
}

// NewResourceUtil returns a new instance of ResourceUtil.
func NewResourceUtil(instanceID string, awsClient *aws.Client) *ResourceUtil {
	return &ResourceUtil{
		awsClient:  awsClient,
		instanceID: instanceID,
	}
}

// GetFilestore returns the Filestore interface that matches the installation.
func (r *ResourceUtil) GetFilestore(installation *model.Installation) model.Filestore {
	switch installation.Filestore {
	case model.InstallationFilestoreMinioOperator:
		return model.NewMinioOperatorFilestore()
	case model.InstallationFilestoreAwsS3:
		return aws.NewS3Filestore(installation.ID, r.awsClient)
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
	case model.InstallationDatabaseSingleTenantRDSMySQL:
		return aws.NewRDSDatabase(model.DatabaseEngineTypeMySQL, installationID, r.awsClient)
	case model.InstallationDatabaseSingleTenantRDSPostgres:
		return aws.NewRDSDatabase(model.DatabaseEngineTypePostgres, installationID, r.awsClient)
	case model.InstallationDatabaseMultiTenantRDSMySQL:
		return aws.NewRDSMultitenantDatabase(model.DatabaseEngineTypeMySQL, r.instanceID, installationID, r.awsClient)
	case model.InstallationDatabaseMultiTenantRDSPostgres:
		return aws.NewRDSMultitenantDatabase(model.DatabaseEngineTypePostgres, r.instanceID, installationID, r.awsClient)
	case model.InstallationDatabaseMultiTenantRDSPostgresPGBouncer:
		return aws.NewRDSMultitenantPGBouncerDatabase(model.DatabaseEngineTypePostgresProxy, r.instanceID, installationID, r.awsClient)
	}

	// Warning: we should never get here as it would mean that we didn't match
	// our database type.
	return model.NewMysqlOperatorDatabase()
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
