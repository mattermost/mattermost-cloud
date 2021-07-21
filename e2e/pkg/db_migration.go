// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//+build e2e

package pkg

import (
	"fmt"
	"time"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-cloud/model"
)

// WaitForDBMigrationToFinish waits for DB migration to reach state Succeeded.
func WaitForDBMigrationToFinish(client *model.Client, opID string, log logrus.FieldLogger) error {
	err := WaitForFunc(NewWaitConfig(20*time.Minute, 10*time.Second, 2, log), func() (bool, error) {
		operation, err := client.GetInstallationDBMigrationOperation(opID)
		if err != nil {
			return false, errors.Wrap(err, "while waiting for db migration")
		}

		if operation.State == model.InstallationDBMigrationStateSucceeded {
			return true, nil
		}
		if operation.State == model.InstallationDBMigrationStateFailed {
			return false, fmt.Errorf("db migration operation %q failed", operation.ID)
		}

		log.Infof("DB migration %s not finished: %s", opID, operation.State)
		return false, nil
	})
	return err
}

// WaitForDBMigrationRollbackToFinish waits for DB migration to reach state RollbackFinished.
func WaitForDBMigrationRollbackToFinish(client *model.Client, opID string, log logrus.FieldLogger) error {
	err := WaitForFunc(NewWaitConfig(10*time.Minute, 10*time.Second, 2, log), func() (bool, error) {
		operation, err := client.GetInstallationDBMigrationOperation(opID)
		if err != nil {
			return false, errors.Wrap(err, "while waiting for db migration rollback")
		}

		if operation.State == model.InstallationDBMigrationStateRollbackFinished {
			return true, nil
		}

		log.Infof("DB migration rollback %s not finished: %s", opID, operation.State)
		return false, nil
	})
	return err
}
