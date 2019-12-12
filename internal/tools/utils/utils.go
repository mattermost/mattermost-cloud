package utils

import (
	"fmt"
	"io"
	"os"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
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

// GetFilestore returns the Filestore interface that matches the installation.
func GetFilestore(i *model.Installation) model.Filestore {
	switch i.Filestore {
	case model.InstallationFilestoreMinioOperator:
		return model.NewMinioOperatorFilestore()
	case model.InstallationFilestoreAwsS3:
		return aws.NewS3Filestore(i.ID)
	}

	// Warning: we should never get here as it would mean that we didn't match
	// our filestore type.
	return model.NewMinioOperatorFilestore()
}

// GetDatabase returns the Database interface that matches the installation.
func GetDatabase(i *model.Installation) model.Database {
	switch i.Database {
	case model.InstallationDatabaseMysqlOperator:
		return model.NewMysqlOperatorDatabase()
	case model.InstallationDatabaseAwsRDS:
		return aws.NewRDSDatabase(i.ID)
	}

	// Warning: we should never get here as it would mean that we didn't match
	// our database type.
	return model.NewMysqlOperatorDatabase()
}
