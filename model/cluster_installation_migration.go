package model

import (
	"encoding/json"
	"io"
)

// ClusterInstallationMigration represents a installation migration.
type ClusterInstallationMigration struct {
	ID                    string
	ClusterID             string
	ClusterInstallationID string
	State                 string
	CreateAt              int64
	DeleteAt              int64
	LockAcquiredBy        *string
	LockAcquiredAt        string
}

// ClusterInstallationMigrationFromReader decodes a json-encoded migration from the given io.Reader.
func ClusterInstallationMigrationFromReader(reader io.Reader) (*ClusterInstallationMigration, error) {
	clusterInstallationMigration := ClusterInstallationMigration{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&clusterInstallationMigration)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &clusterInstallationMigration, nil
}

// ClusterInstallationMigrationFilter describes the parameters used to constrain a set of cluster installations migrations.
type ClusterInstallationMigrationFilter struct {
	Page           int
	PerPage        int
	IncludeDeleted bool
}
