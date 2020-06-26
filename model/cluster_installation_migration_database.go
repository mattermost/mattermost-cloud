// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	log "github.com/sirupsen/logrus"
)

const (
	// DatabaseMigrationStatusSetupIP indicates that database migration setup is still running.
	DatabaseMigrationStatusSetupIP = "setup-in-progress"
	// DatabaseMigrationStatusSetupComplete indicates that database migration setup is completed.
	DatabaseMigrationStatusSetupComplete = "setup-complete"
	// DatabaseMigrationStatusTeardownIP indicates that database migration teardown is still running.
	DatabaseMigrationStatusTeardownIP = "teardown-in-progress"
	// DatabaseMigrationStatusTeardownComplete indicates that database migration teardown is completed.
	DatabaseMigrationStatusTeardownComplete = "teardown-complete"
	// DatabaseMigrationStatusReplicationIP indicates that database migration replication process is still running.
	DatabaseMigrationStatusReplicationIP = "replication-in-progress"
	// DatabaseMigrationStatusReplicationComplete indicates that database migration process is completed.
	DatabaseMigrationStatusReplicationComplete = "replication-complete"
)

// CIMigrationDatabase is the interface for managing Mattermost databases migration process.
type CIMigrationDatabase interface {
	Setup(logger log.FieldLogger) (string, error)
	Teardown(logger log.FieldLogger) (string, error)
	Replicate(logger log.FieldLogger) (string, error)
}
