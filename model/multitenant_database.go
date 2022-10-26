// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

// defaultProxyDatabaseMaxInstallationsPerLogicalDatabase is the default value
// used for MaxInstallationsPerLogicalDatabase when new multitenant databases
// are created.
var defaultProxyDatabaseMaxInstallationsPerLogicalDatabase int64 = 10

// SetDefaultProxyDatabaseMaxInstallationsPerLogicalDatabase is used to define
// a new value for defaultProxyDatabaseMaxInstallationsPerLogicalDatabase.
func SetDefaultProxyDatabaseMaxInstallationsPerLogicalDatabase(val int64) error {
	if val < 1 {
		return errors.New("MaxInstallationsPerLogicalDatabase must be set to 1 or higher")
	}
	defaultProxyDatabaseMaxInstallationsPerLogicalDatabase = val

	return nil
}

// GetDefaultProxyDatabaseMaxInstallationsPerLogicalDatabase returns the value
// of defaultProxyDatabaseMaxInstallationsPerLogicalDatabase.
func GetDefaultProxyDatabaseMaxInstallationsPerLogicalDatabase() int64 {
	return defaultProxyDatabaseMaxInstallationsPerLogicalDatabase
}

// MultitenantDatabase represents database infrastructure that contains multiple
// installation databases.
type MultitenantDatabase struct {
	ID                                 string
	RdsClusterID                       string
	VpcID                              string
	DatabaseType                       string
	State                              string
	WriterEndpoint                     string
	ReaderEndpoint                     string
	Installations                      MultitenantDatabaseInstallations
	MigratedInstallations              MultitenantDatabaseInstallations
	MaxInstallationsPerLogicalDatabase int64 `json:"MaxInstallationsPerLogicalDatabase,omitempty"`
	CreateAt                           int64
	DeleteAt                           int64
	LockAcquiredBy                     *string
	LockAcquiredAt                     int64
}

// CreationDateString returns a standardized date string for a multitenant
// database string.
func (d *MultitenantDatabase) CreationDateString() string {
	return GetDateString(d.CreateAt)
}

// LogicalDatabase represents a logical database inside a MultitenantDatabase.
type LogicalDatabase struct {
	ID                    string
	MultitenantDatabaseID string
	Name                  string
	CreateAt              int64
	DeleteAt              int64
	LockAcquiredBy        *string
	LockAcquiredAt        int64
}

// DatabaseSchema represents a database schema inside a LogicalDatabase.
type DatabaseSchema struct {
	ID                string
	LogicalDatabaseID string
	InstallationID    string
	Name              string
	CreateAt          int64
	DeleteAt          int64
	LockAcquiredBy    *string
	LockAcquiredAt    int64
}

// DatabaseResourceGrouping represents the complete set of database resources
// that comprise proxy database information.
type DatabaseResourceGrouping struct {
	MultitenantDatabase *MultitenantDatabase
	LogicalDatabase     *LogicalDatabase
	DatabaseSchema      *DatabaseSchema
}

// GetReaderEndpoint returns the best available reader endpoint for a multitenant
// database.
func (d *MultitenantDatabase) GetReaderEndpoint() string {
	if len(d.ReaderEndpoint) != 0 {
		return d.ReaderEndpoint
	}

	return d.WriterEndpoint
}

// MultitenantDatabaseInstallations is the list of installation IDs that belong
// to a given MultitenantDatabase.
type MultitenantDatabaseInstallations []string

// Count returns the number of installations on the multitenant database.
func (i *MultitenantDatabaseInstallations) Count() int {
	return len(*i)
}

// Contains checks if the supplied installation ID exists in the container.
func (i *MultitenantDatabaseInstallations) Contains(installationID string) bool {
	for _, id := range *i {
		if id == installationID {
			return true
		}
	}

	return false
}

// Add inserts a new installation in the container.
func (i *MultitenantDatabaseInstallations) Add(installationID string) {
	*i = append(*i, installationID)
}

// Remove deletes the installation from the container.
func (i *MultitenantDatabaseInstallations) Remove(installationID string) {
	for j, installation := range *i {
		if installation == installationID {
			(*i) = append((*i)[:j], (*i)[j+1:]...)
		}
	}
}

// MultitenantDatabaseFilter describes the parameters used to constrain a set of
// MultitenantDatabases.
type MultitenantDatabaseFilter struct {
	Paging
	LockerID               string
	InstallationID         string
	MigratedInstallationID string
	VpcID                  string
	DatabaseType           string
	MaxInstallationsLimit  int
}

// LogicalDatabaseFilter describes the parameters used to constrain a set of
// LogicalDatabase.
type LogicalDatabaseFilter struct {
	Paging
	MultitenantDatabaseID string
}

// DatabaseSchemaFilter describes the parameters used to constrain a set of
// DatabaseSchema.
type DatabaseSchemaFilter struct {
	Paging
	LogicalDatabaseID string
	InstallationID    string
}

// MultitenantDatabasesFromReader decodes a json-encoded list of multitenant databases from the given io.Reader.
func MultitenantDatabasesFromReader(reader io.Reader) ([]*MultitenantDatabase, error) {
	multitenantDatabases := []*MultitenantDatabase{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&multitenantDatabases)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return multitenantDatabases, nil
}

// MultitenantDatabaseFromReader decodes a json-encoded multitenant database from the given io.Reader.
func MultitenantDatabaseFromReader(reader io.Reader) (*MultitenantDatabase, error) {
	multitenantDatabase := &MultitenantDatabase{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&multitenantDatabase)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return multitenantDatabase, nil
}

// LogicalDatabasesFromReader decodes a json-encoded list of logical databases from the given io.Reader.
func LogicalDatabasesFromReader(reader io.Reader) ([]*LogicalDatabase, error) {
	logicalDatabases := []*LogicalDatabase{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&logicalDatabases)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return logicalDatabases, nil
}

// LogicalDatabaseFromReader decodes a json-encoded logical database from the given io.Reader.
func LogicalDatabaseFromReader(reader io.Reader) (*LogicalDatabase, error) {
	logicalDatabase := &LogicalDatabase{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&logicalDatabase)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return logicalDatabase, nil
}

// DatababseSchemasFromReader decodes a json-encoded list of database schemas from the given io.Reader.
func DatababseSchemasFromReader(reader io.Reader) ([]*DatabaseSchema, error) {
	databaseSchemas := []*DatabaseSchema{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&databaseSchemas)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return databaseSchemas, nil
}

// DatababseSchemaFromReader decodes a json-encoded database schema from the given io.Reader.
func DatababseSchemaFromReader(reader io.Reader) (*DatabaseSchema, error) {
	databaseSchema := &DatabaseSchema{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&databaseSchema)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return databaseSchema, nil
}
