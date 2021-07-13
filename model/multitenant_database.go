// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"fmt"
	"io"
)

// MultitenantDatabase represents database infrastructure that contains multiple
// installation databases.
type MultitenantDatabase struct {
	ID                                 string
	VpcID                              string
	DatabaseType                       string
	State                              string
	WriterEndpoint                     string
	ReaderEndpoint                     string
	Installations                      MultitenantDatabaseInstallations
	MigratedInstallations              MultitenantDatabaseInstallations
	SharedLogicalDatabaseMappings      SharedLogicalDatabases `json:"SharedLogicalDatabaseMappings,omitempty"`
	MaxInstallationsPerLogicalDatabase int64                  `json:"MaxInstallationsPerLogicalDatabase,omitempty"`
	CreateAt                           int64
	DeleteAt                           int64
	LockAcquiredBy                     *string
	LockAcquiredAt                     int64
}

// AddInstallationToLogicalDatabaseMapping adds a new installation to the next
// available logical database.
func (d *MultitenantDatabase) AddInstallationToLogicalDatabaseMapping(installationID string) {
	if d.SharedLogicalDatabaseMappings == nil {
		d.SharedLogicalDatabaseMappings = make(SharedLogicalDatabases)
	}

	// Ensure we always are adding installations to the logical databases that
	// are most full for maximum efficiency.
	var selectedLogicalDatabase string
	for logicalDatabase, installations := range d.SharedLogicalDatabaseMappings {
		if len(installations) > int(d.MaxInstallationsPerLogicalDatabase) {
			continue
		}
		if len(selectedLogicalDatabase) == 0 {
			selectedLogicalDatabase = logicalDatabase
			continue
		}
		if len(installations) > len(d.SharedLogicalDatabaseMappings[selectedLogicalDatabase]) {
			selectedLogicalDatabase = logicalDatabase
		}
	}

	if len(selectedLogicalDatabase) == 0 {
		// None of the existing logical databases had room so create a new one with
		// a unique ID.
		d.SharedLogicalDatabaseMappings[fmt.Sprintf("cloud_%s", NewID())] = []string{installationID}
		return
	}

	d.SharedLogicalDatabaseMappings[selectedLogicalDatabase] = append(d.SharedLogicalDatabaseMappings[selectedLogicalDatabase], installationID)
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

// SharedLogicalDatabases is a mapping of logical databases to installations.
type SharedLogicalDatabases map[string][]string

// GetLogicalDatabaseName returns the logical database that an installation
// belongs to or an empty string if it hasn't been assigned.
func (l *SharedLogicalDatabases) GetLogicalDatabaseName(installationID string) string {
	for logicalDatabase, installations := range *l {
		for _, installation := range installations {
			if installation == installationID {
				return logicalDatabase
			}
		}
	}

	return ""
}

// RemoveInstallation removes an installation entry from the logical database
// mapping.
func (l *SharedLogicalDatabases) RemoveInstallation(installationID string) {
	for logicalDatabase, installations := range *l {
		for i, installation := range installations {
			if installation == installationID {
				(*l)[logicalDatabase] = append(installations[:i], installations[i+1:]...)
				return
			}
		}
	}
}

// MultitenantDatabaseFilter filters results based on a specific installation ID, Vpc ID and a number of
// installation's limit.
type MultitenantDatabaseFilter struct {
	Paging
	LockerID               string
	InstallationID         string
	MigratedInstallationID string
	VpcID                  string
	DatabaseType           string
	MaxInstallationsLimit  int
}

// MultitenantDatabasesFromReader decodes a json-encoded list of multitenant databases from the given io.Reader.
func MultitenantDatabasesFromReader(reader io.Reader) ([]*MultitenantDatabase, error) {
	databases := []*MultitenantDatabase{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&databases)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return databases, nil
}

// MultitenantDatabaseFromReader decodes a json-encoded multitenant database from the given io.Reader.
func MultitenantDatabaseFromReader(reader io.Reader) (*MultitenantDatabase, error) {
	database := &MultitenantDatabase{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&database)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return database, nil
}
