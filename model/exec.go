// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

const (
	execMMCTL         = "mmctl"
	execMattermostCLI = "mattermost"
)

// IsValidExecCommand returns wheather the provided command is valid or not.
func IsValidExecCommand(command string) bool {
	switch command {
	case execMMCTL, execMattermostCLI:
		return true
	}

	return false
}
