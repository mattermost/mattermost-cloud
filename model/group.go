// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"

	log "github.com/sirupsen/logrus"
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
	Annotations *AnnotationsFilter

	// WithInstallationCount if the store should retrieve the count of non-deleted installations
	// for this group
	WithInstallationCount bool
}

// ToDTO returns Group joined with Annotations.
func (g *Group) ToDTO(annotations []*Annotation) *GroupDTO {
	return &GroupDTO{
		Group:       g,
		Annotations: annotations,
	}
}

// Clone returns a deep copy the group.
func (g *Group) Clone() *Group {
	var clone Group
	data, _ := json.Marshal(g)
	if err := json.Unmarshal(data, &clone); err != nil {
		log.WithError(err).Error("failed to unmarshal group clone")
	}
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
