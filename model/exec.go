// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

const execMMCTL = "mmctl"

// IsValidExecCommand returns wheather the provided command is valid or not.
func IsValidExecCommand(command string) bool {
	return command == execMMCTL
}
