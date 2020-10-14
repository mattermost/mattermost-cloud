// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
)

// GroupStatus represents the status of a group.
type GroupStatus struct {
	InstallationsTotal          int64
	InstallationsUpdated        int64
	InstallationsBeingUpdated   int64
	InstallationsAwaitingUpdate int64
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
