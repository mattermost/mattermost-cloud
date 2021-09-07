// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

// This package contains structs used in unit tests to avoid loading test packages in the generator.

package testdata

// TestStruct is a struct for the purpose of generator unit tests.
type TestStruct struct {
	ID       string
	State    string
	DeleteAt int64
}
