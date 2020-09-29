// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
)

// GroupStatus represents the status of group rollout
type GroupStatus struct {
	InstallationsCount           int64
	InstallationsRolledOut       int64
	InstallationsAwaitingRollOut int64
}

// GroupStatusFromReader decodes a json-encoded group status from the given io.Reader.
func GroupStatusFromReader(reader io.Reader) (*GroupStatus, error) {
	groupStatus := GroupStatus{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&groupStatus)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &groupStatus, nil
}
