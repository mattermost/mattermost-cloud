// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

const (
	// AllPerPage signals the store to return all results, avoid pagination of any kind.
	AllPerPage = -1

	// NoInstallationsLimit signals the store to return all multitenant database instances independently
	// of the number of installations using each instance.
	NoInstallationsLimit = -1

	// NonApplicableState represents constant use for StateChangeEvents when resource did not have a previous state.
	NonApplicableState = "n/a"
)
