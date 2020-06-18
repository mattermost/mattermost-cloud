package model

import (
	"encoding/json"
	"io"
)

// Group represents a group of Mattermost installations.
type Group struct {
	ID             string    `json:"id,omitempty"`
	Sequence       int64     `json:"sequence,omitempty"`
	Name           string    `json:"name,omitempty"`
	Description    string    `json:"description,omitempty"`
	Version        string    `json:"version,omitempty"`
	Image          string    `json:"image,omitempty"`
	MaxRolling     int64     `json:"maxRolling,omitempty"`
	MattermostEnv  EnvVarMap `json:"mattermostEnv,omitempty"`
	CreateAt       int64     `json:"createAt,omitempty"`
	DeleteAt       int64     `json:"deleteAt,omitempty"`
	LockAcquiredBy *string   `json:"lockAcquiredBy,omitempty"`
	LockAcquiredAt int64     `json:"lockAcquiredAt,omitempty"`
}

// GroupFilter describes the parameters used to constrain a set of groups.
type GroupFilter struct {
	Page           int
	PerPage        int
	IncludeDeleted bool
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
