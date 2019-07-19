package model

import (
	"encoding/json"
	"io"
)

// Group represents a group of Mattermost installations.
type Group struct {
	ID          string
	Name        string
	Description string
	Version     string
	CreateAt    int64
	DeleteAt    int64
}

// GroupFilter describes the parameters used to constrain a set of groups.
type GroupFilter struct {
	Page           int
	PerPage        int
	IncludeDeleted bool
}

// Clone returns a deep copy the group.
func (c *Group) Clone() *Group {
	var clone Group
	data, _ := json.Marshal(c)
	json.Unmarshal(data, &clone)

	return &clone
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
