package model

// CMI = Cluster Installation Migration

const (
	// CIMigrationStable is an InstallationMigration in a stable state and undergoing no changes.
	CIMigrationStable = "stable"

	// CIMigrationCreationRequested ...
	CIMigrationCreationRequested = "creation-requested"
	// CIMigrationCreationComplete ...
	CIMigrationCreationComplete = "creation-complete"
	// CIMigrationCreationFailed ...
	CIMigrationCreationFailed = "creation-failed"
	// CIMigrationSnapshotCreationComplete indicates that the snapshot creation has started.
	CIMigrationSnapshotCreationComplete = "snapshot-creation-complete"
	// CIMigrationRestoreDatabaseComplete indicates that a database is being restored.
	CIMigrationRestoreDatabaseComplete = "restore-database-complete"
	// CIMigrationSetupDatabaseComplete indicates that a database has been configured.
	CIMigrationSetupDatabaseComplete = "restore-setup-complete"

	// CIMigrationClusterInstallationCreationComplete indicates that a new cluster installation creation has started.
	CIMigrationClusterInstallationCreationComplete = "cluster-installation-creation-complete"

	// CIMigrationCreationInProgress is an InstallationMigration in the process of being created.
	// CIMigrationCreationInProgress = "creation-in-progress"
)

// AllCIMigrations is a list of all states an InstallationMigration can be in.
// Warning:
// When creating a new InstallationMigration state, it must be added to this list.
var AllCIMigrations = []string{
	CIMigrationStable,
	CIMigrationCreationRequested,
	CIMigrationCreationComplete,
	CIMigrationCreationFailed,
	CIMigrationSnapshotCreationComplete,
	CIMigrationRestoreDatabaseComplete,
	CIMigrationSetupDatabaseComplete,
}

// AllCIMigrationsPendingWork is a list of all InstallationMigration states that
// the supervisor will attempt to transition towards stable on the next "tick".
// Warning:
// When creating a new InstallationMigration state, it must be added to this list if the
// cloud InstallationMigration supervisor should perform some action on its next work cycle.
var AllCIMigrationsPendingWork = []string{
	CIMigrationCreationRequested,
	CIMigrationCreationComplete,
	CIMigrationRestoreDatabaseComplete,
	CIMigrationSetupDatabaseComplete,
}

// AllCMIRequestStates is a list of all states that an InstallationMigration can
// be put in via the API.
// Warning:
// When creating a new InstallationMigration state, it must be added to this list if an
// API endpoint should put the InstallationMigration in this state.
var AllCMIRequestStates = []string{
	CIMigrationCreationRequested,
}
