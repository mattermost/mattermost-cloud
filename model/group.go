// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
)

// Group represents a group of Mattermost installations.
type Group struct {
	ID              string
	Sequence        int64
	Name            string
	Description     string
	Version         string
	Image           string
	MaxRolling      int64
	MattermostEnv   EnvVarMap
	CreateAt        int64
	DeleteAt        int64
	APISecurityLock bool
	LockAcquiredBy  *string
	LockAcquiredAt  int64
}

// GroupFilter describes the parameters used to constrain a set of groups.
type GroupFilter struct {
	Paging
}

// Clone returns a deep copy the group.
func (g *Group) Clone() *Group {
	var clone Group
	data, _ := json.Marshal(g)
	json.Unmarshal(data, &clone)

	return &clone
}

// IsDeleted returns whether the group is deleted or not.
func (g *Group) IsDeleted() bool {
	return g.DeleteAt != 0
}

// GroupFromReader decodes a json-encoded group from the given io.Reader.
func GroupFromReader(reader io.Reader) (*Group, error) {
	group := Group{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&group)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &group, nil
}

// GroupsFromReader decodes a json-encoded list of groups from the given io.Reader.
func GroupsFromReader(reader io.Reader) ([]*Group, error) {
	groups := []*Group{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&groups)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return groups, nil
}
