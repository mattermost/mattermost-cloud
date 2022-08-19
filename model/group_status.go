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
	InstallationsTotal           int64
	InstallationsUpdated         int64
	InstallationsUpdating        int64
	InstallationsHibernating     int64
	InstallationsPendingDeletion int64
	InstallationsAwaitingUpdate  int64
}

// GroupsStatus represents the status of a groups.
type GroupsStatus struct {
	ID     string
	Status GroupStatus
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

// GroupsStatusFromReader decodes a json-encoded groups status from the given io.Reader.
func GroupsStatusFromReader(reader io.Reader) ([]*GroupsStatus, error) {
	groupsStatus := []*GroupsStatus{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&groupsStatus)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return groupsStatus, nil
}
