// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
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
	Scheduling      *Scheduling `json:"Scheduling,omitempty"`
	CreateAt        int64
	DeleteAt        int64
	APISecurityLock bool
	LockAcquiredBy  *string
	LockAcquiredAt  int64
}

// Scheduling contains configuration for overriding pod scheduling settings.
type Scheduling struct {
	NodeSelector map[string]string
	Tolerations  []corev1.Toleration
}

// Value implements the driver.Valuer interface for database storage
func (s *Scheduling) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan implements the sql.Scanner interface for database retrieval
func (s *Scheduling) Scan(src interface{}) error {
	if src == nil {
		return nil
	}

	source, ok := src.([]byte)
	if !ok {
		return errors.New("could not assert type of Scheduling")
	}

	var override Scheduling
	err := json.Unmarshal(source, &override)
	if err != nil {
		return err
	}
	*s = override
	return nil
}

// NodeSelectorValueString returns the string value of the NodeSelector.
func (s *Scheduling) NodeSelectorValueString() string {
	if s.NodeSelector == nil {
		return "null"
	}
	var kvPairs []string
	for k, v := range s.NodeSelector {
		kvPairs = append(kvPairs, fmt.Sprintf("%s=%s", k, v))
	}
	return strings.Join(kvPairs, ", ")
}

// TolerationsValueString returns the string value of the Tolerations.
func (s *Scheduling) TolerationsValueString() string {
	if s.Tolerations == nil {
		return "null"
	}
	var tolerations []string
	for _, toleration := range s.Tolerations {
		tolerations = append(tolerations, toleration.String())
	}
	return strings.Join(tolerations, ", ")
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
