// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

// MultitenantDatabase represents database infrastructure that contains multiple
// installation databases.
type MultitenantDatabase struct {
	ID             string
	VpcID          string
	DatabaseType   string
	Installations  MultitenantDatabaseInstallations
	LockAcquiredBy *string
	CreateAt       int64
	DeleteAt       int64
	LockAcquiredAt int64
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

// MultitenantDatabaseFilter filters results based on a specific installation ID, Vpc ID and a number of
// installation's limit.
type MultitenantDatabaseFilter struct {
	LockerID              string
	InstallationID        string
	VpcID                 string
	DatabaseType          string
	MaxInstallationsLimit int
	Page                  int
	PerPage               int
}
