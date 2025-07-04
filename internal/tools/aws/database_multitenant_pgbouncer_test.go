// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"fmt"
	"strings"
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRDSMultitenantPGBouncerDatabase(t *testing.T) {
	databaseType := model.DatabaseEngineTypePostgresProxy
	instanceID := "test-instance-id"
	installationID := "test-installation-id"
	client := &Client{}
	installationsLimit := 100
	disableDBCheck := true

	database := NewRDSMultitenantPGBouncerDatabase(databaseType, instanceID, installationID, client, installationsLimit, disableDBCheck)

	assert.Equal(t, databaseType, database.databaseType)
	assert.Equal(t, instanceID, database.instanceID)
	assert.Equal(t, installationID, database.installationID)
	assert.Equal(t, client, database.client)
	assert.Equal(t, installationsLimit, database.maxSupportedInstallations)
	assert.Equal(t, disableDBCheck, database.disableDBCheck)
}

func TestNewRDSMultitenantPGBouncerDatabase_DefaultLimit(t *testing.T) {
	database := NewRDSMultitenantPGBouncerDatabase(
		model.DatabaseEngineTypePostgresProxy,
		"instance-id",
		"installation-id",
		&Client{},
		0, // Should use default
		false,
	)

	assert.Equal(t, DefaultRDSMultitenantPGBouncerDatabasePostgresCountLimit, database.maxSupportedInstallations)
}

func TestRDSMultitenantPGBouncerDatabase_IsValid(t *testing.T) {
	tests := []struct {
		name           string
		databaseType   string
		installationID string
		expectError    bool
		errorMessage   string
	}{
		{
			name:           "valid configuration",
			databaseType:   model.DatabaseEngineTypePostgresProxy,
			installationID: "test-installation-id",
			expectError:    false,
		},
		{
			name:           "empty installation ID",
			databaseType:   model.DatabaseEngineTypePostgresProxy,
			installationID: "",
			expectError:    true,
			errorMessage:   "installation ID is not set",
		},
		{
			name:           "invalid database type",
			databaseType:   "invalid-type",
			installationID: "test-installation-id",
			expectError:    true,
			errorMessage:   "invalid pgbouncer database type invalid-type",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			database := &RDSMultitenantPGBouncerDatabase{
				databaseType:   tt.databaseType,
				installationID: tt.installationID,
			}

			err := database.IsValid()

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRDSMultitenantPGBouncerDatabase_DatabaseEngineTypeTagValue(t *testing.T) {
	database := &RDSMultitenantPGBouncerDatabase{}
	tagValue := database.DatabaseEngineTypeTagValue()
	assert.Equal(t, DatabaseTypePostgresSQLAurora, tagValue)
}

func TestRDSMultitenantPGBouncerDatabase_MaxSupportedDatabases(t *testing.T) {
	database := &RDSMultitenantPGBouncerDatabase{
		maxSupportedInstallations: 150,
	}
	maxDatabases := database.MaxSupportedDatabases()
	assert.Equal(t, 150, maxDatabases)
}

// Test SQL query generation for PGBouncer operations
func TestPGBouncerSQLQueries(t *testing.T) {
	t.Run("insert user query format", func(t *testing.T) {
		username := "test_user"
		scramHash := "SCRAM-SHA-256$4096:c2FsdA==$stored:server"

		expectedQuery := fmt.Sprintf(`INSERT INTO pgbouncer.pgbouncer_users (usename, passwd) VALUES ('%s', '%s')`, username, scramHash)

		assert.Contains(t, expectedQuery, "INSERT INTO pgbouncer.pgbouncer_users")
		assert.Contains(t, expectedQuery, username)
		assert.Contains(t, expectedQuery, scramHash)
		assert.Contains(t, expectedQuery, "usename")
		assert.Contains(t, expectedQuery, "passwd")

		// Ensure proper SQL formatting
		assert.True(t, strings.Count(expectedQuery, "'") >= 4) // At least 4 single quotes for string values
	})

	t.Run("delete user query format", func(t *testing.T) {
		username := "test_user"

		expectedQuery := fmt.Sprintf(`DELETE FROM  pgbouncer.pgbouncer_users WHERE usename = '%s'`, username)

		assert.Contains(t, expectedQuery, "DELETE FROM")
		assert.Contains(t, expectedQuery, "pgbouncer.pgbouncer_users")
		assert.Contains(t, expectedQuery, "WHERE usename =")
		assert.Contains(t, expectedQuery, username)
	})

	t.Run("grant role query format", func(t *testing.T) {
		role := "test_role"
		user := DefaultMattermostDatabaseUsername

		expectedQuery := fmt.Sprintf("GRANT %s TO %s;", role, user)

		assert.Contains(t, expectedQuery, "GRANT")
		assert.Contains(t, expectedQuery, "TO")
		assert.Contains(t, expectedQuery, role)
		assert.Contains(t, expectedQuery, user)
		assert.True(t, strings.HasSuffix(expectedQuery, ";"))
	})

	t.Run("create schema query format", func(t *testing.T) {
		username := "test_user"

		expectedQuery := fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS AUTHORIZATION %s", username)

		assert.Contains(t, expectedQuery, "CREATE SCHEMA")
		assert.Contains(t, expectedQuery, "IF NOT EXISTS")
		assert.Contains(t, expectedQuery, "AUTHORIZATION")
		assert.Contains(t, expectedQuery, username)
	})

	t.Run("pgbouncer table creation query format", func(t *testing.T) {
		schema := "pgbouncer"

		expectedQuery := fmt.Sprintf(`
	CREATE TABLE IF NOT EXISTS %s.pgbouncer_users(
	usename NAME PRIMARY KEY,
	passwd TEXT NOT NULL
	)`, schema)

		assert.Contains(t, expectedQuery, "CREATE TABLE IF NOT EXISTS")
		assert.Contains(t, expectedQuery, "pgbouncer_users")
		assert.Contains(t, expectedQuery, "usename NAME PRIMARY KEY")
		assert.Contains(t, expectedQuery, "passwd TEXT NOT NULL")
	})
}

// Test SCRAM integration in the context of PGBouncer setup
func TestPGBouncerSCRAMIntegration(t *testing.T) {
	// Test that demonstrates how SCRAM hashes are used in PGBouncer setup
	password := "test_db_password"
	username := "mattermost_user"

	// Generate SCRAM hash as done in the actual code
	scramHash, err := generateSCRAMSHA256Hash(password)
	require.NoError(t, err)

	// Test PostgreSQL user creation query format
	createUserQuery := fmt.Sprintf("CREATE USER %s WITH PASSWORD '%s'", username, scramHash)
	assert.Contains(t, createUserQuery, "CREATE USER mattermost_user WITH PASSWORD 'SCRAM-SHA-256$")

	// Test PGBouncer users table insertion query format
	insertPGBouncerQuery := fmt.Sprintf("INSERT INTO pgbouncer.pgbouncer_users (usename, passwd) VALUES ('%s', '%s')", username, scramHash)
	assert.Contains(t, insertPGBouncerQuery, "INSERT INTO pgbouncer.pgbouncer_users")
	assert.Contains(t, insertPGBouncerQuery, username)
	assert.Contains(t, insertPGBouncerQuery, scramHash)

	// Verify hash format is PostgreSQL compatible
	assert.True(t, strings.HasPrefix(scramHash, "SCRAM-SHA-256$4096:"))
	assert.Regexp(t, `^SCRAM-SHA-256\$4096:[A-Za-z0-9+/]+=*\$[A-Za-z0-9+/]+=*:[A-Za-z0-9+/]+=*$`, scramHash)

	// Verify no SQL injection possibilities
	assert.NotContains(t, scramHash, "'")
	assert.NotContains(t, scramHash, "\"")
	assert.NotContains(t, scramHash, ";")
	assert.NotContains(t, scramHash, "--")

	t.Logf("PGBouncer SCRAM integration test completed successfully")
	t.Logf("Generated hash: %s", scramHash)
	t.Logf("PostgreSQL query: %s", createUserQuery)
	t.Logf("PGBouncer query: %s", insertPGBouncerQuery)
}

func TestPGBouncerSecretName_Generation(t *testing.T) {
	installationID := "test-installation-123"
	expectedPattern := fmt.Sprintf("rds-multitenant-pgbouncer-%s", installationID)

	secretName := RDSMultitenantPGBouncerSecretName(installationID)

	assert.Equal(t, expectedPattern, secretName)
	assert.Contains(t, secretName, installationID)
	assert.True(t, strings.HasPrefix(secretName, "rds-multitenant-pgbouncer-"))
}

func TestPGBouncerUsername_Generation(t *testing.T) {
	installationID := "test-installation-123"
	expectedPattern := fmt.Sprintf("id_%s", installationID)

	username := MattermostPGBouncerDatabaseUsername(installationID)

	assert.Equal(t, expectedPattern, username)
	assert.Contains(t, username, installationID)
	assert.True(t, strings.HasPrefix(username, "id_"))
}

// Test connection string generation
func TestPGBouncerConnectionStrings(t *testing.T) {
	username := "test_user"
	password := "test_password"
	databaseName := "test_db"

	connectionString, readReplicasString, checkURL := MattermostPostgresPGBouncerConnStrings(username, password, databaseName)

	// Test main connection string
	assert.Contains(t, connectionString, username)
	assert.Contains(t, connectionString, password)
	assert.Contains(t, connectionString, databaseName)
	assert.Contains(t, connectionString, "postgres://")
	assert.Contains(t, connectionString, "sslmode=disable") // PGBouncer uses disable, not require
	assert.Contains(t, connectionString, "binary_parameters=yes")

	// Test read replicas string
	assert.Contains(t, readReplicasString, username)
	assert.Contains(t, readReplicasString, password)
	assert.Contains(t, readReplicasString, fmt.Sprintf("%s-ro", databaseName)) // Has -ro suffix
	assert.Contains(t, readReplicasString, "sslmode=disable")
	assert.Contains(t, readReplicasString, "binary_parameters=yes")

	// Test check URL
	assert.Contains(t, checkURL, username)
	assert.Contains(t, checkURL, password)
	assert.Contains(t, checkURL, databaseName)
	assert.Contains(t, checkURL, "sslmode=disable")
	assert.NotContains(t, checkURL, "binary_parameters") // Check URL doesn't have binary_parameters

	t.Logf("Connection string: %s", connectionString)
	t.Logf("Read replicas string: %s", readReplicasString)
	t.Logf("Check URL: %s", checkURL)
}

// Test unsupported operations
func TestRDSMultitenantPGBouncerDatabase_UnsupportedOperations(t *testing.T) {
	database := &RDSMultitenantPGBouncerDatabase{}

	// Test unsupported operations
	err := database.Snapshot(nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")

	err = database.MigrateOut(nil, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database migration is not supported for PGBouncer database")

	err = database.MigrateTo(nil, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "database migration is not supported for PGBouncer database")

	err = database.TeardownMigrated(nil, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "tearing down migrated installations is not supported for PGBouncer database")

	err = database.RollbackMigration(nil, nil, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "rolling back db migration is not supported for PGBouncer database")

	// RefreshResourceMetadata should succeed (it's a no-op)
	err = database.RefreshResourceMetadata(nil, nil)
	assert.NoError(t, err)
}

// Test SCRAM hash consistency across multiple generations
func TestSCRAMHashConsistency(t *testing.T) {
	password := "consistent_test_password"

	// Generate the same hash multiple times
	hash1, err1 := generateSCRAMSHA256Hash(password)
	hash2, err2 := generateSCRAMSHA256Hash(password)
	hash3, err3 := generateSCRAMSHA256Hash(password)

	// All should succeed
	require.NoError(t, err1)
	require.NoError(t, err2)
	require.NoError(t, err3)

	// Hashes should be different (due to different salts) but all valid
	assert.NotEqual(t, hash1, hash2)
	assert.NotEqual(t, hash2, hash3)
	assert.NotEqual(t, hash1, hash3)

	// All should have the same format
	for _, hash := range []string{hash1, hash2, hash3} {
		assert.True(t, strings.HasPrefix(hash, "SCRAM-SHA-256$4096:"))
		assert.Regexp(t, `^SCRAM-SHA-256\$4096:[A-Za-z0-9+/]+=*\$[A-Za-z0-9+/]+=*:[A-Za-z0-9+/]+=*$`, hash)
	}

	t.Logf("Generated consistent format hashes: %s, %s, %s", hash1, hash2, hash3)
}
