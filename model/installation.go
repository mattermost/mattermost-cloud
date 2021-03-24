// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"fmt"
	"io"
)

const (
	// V1alphaCRVersion is a ClusterInstallation CR alpha version.
	V1alphaCRVersion = "mattermost.com/v1alpha1"
	// V1betaCRVersion is a Mattermost CR beta version.
	V1betaCRVersion = "installation.mattermost.com/v1beta1"
	// DefaultCRVersion is a default CR version used for new installations.
	DefaultCRVersion = V1betaCRVersion
	// DefaultDatabaseWeight is the default weight of a small or average-sized
	// installation that isn't hibernating.
	DefaultDatabaseWeight float64 = 1
	// HibernatingDatabaseWeight is the weight of a hibernating installation.
	HibernatingDatabaseWeight float64 = .75
)

// Installation represents a Mattermost installation.
type Installation struct {
	ID                         string
	OwnerID                    string
	GroupID                    *string
	GroupSequence              *int64 `json:"GroupSequence,omitempty"`
	Version                    string
	Image                      string
	DNS                        string
	Database                   string
	Filestore                  string
	License                    string
	MattermostEnv              EnvVarMap
	Size                       string
	Affinity                   string
	State                      string
	CRVersion                  string
	CreateAt                   int64
	DeleteAt                   int64
	APISecurityLock            bool
	LockAcquiredBy             *string
	LockAcquiredAt             int64
	GroupOverrides             map[string]string           `json:"GroupOverrides,omitempty"`
	SingleTenantDatabaseConfig *SingleTenantDatabaseConfig `json:"SingleTenantDatabaseConfig,omitempty"`

	// configconfigMergedWithGroup is set when the installation configuration
	// has been overridden with group configuration. This value can then be
	// checked later to determine whether the installation is safe to save or
	// not.
	configMergedWithGroup bool

	// configMergeGroupSequence is the Sequence value of the group at the time
	// it was merged with the installation.
	configMergeGroupSequence int64
}

// InstallationsCount represents the number of installations
type InstallationsCount struct {
	Count int64
}

// InstallationFilter describes the parameters used to constrain a set of installations.
type InstallationFilter struct {
	Paging
	InstallationIDs []string
	OwnerID         string
	GroupID         string
	State           string
	DNS             string
}

// Clone returns a deep copy the installation.
func (i *Installation) Clone() *Installation {
	var clone Installation
	data, _ := json.Marshal(i)
	json.Unmarshal(data, &clone)

	return &clone
}

// ToDTO expands installation to InstallationDTO.
func (i *Installation) ToDTO(annotations []*Annotation) *InstallationDTO {
	return &InstallationDTO{
		Installation: i,
		Annotations:  annotations,
	}
}

// GetDatabaseWeight returns a value corresponding to the
// TODO: maybe consider installation size in the future as well?
func (i *Installation) GetDatabaseWeight() float64 {
	if i.State == InstallationStateHibernationRequested ||
		i.State == InstallationStateHibernationInProgress ||
		i.State == InstallationStateHibernating {
		return HibernatingDatabaseWeight
	}

	return DefaultDatabaseWeight
}

// IsInGroup returns if the installation is in a group or not.
func (i *Installation) IsInGroup() bool {
	return i.GroupID != nil
}

// ConfigMergedWithGroup returns if the installation currently has inherited
// group configuration values.
func (i *Installation) ConfigMergedWithGroup() bool {
	return i.configMergedWithGroup
}

// InstallationSequenceMatchesMergedGroupSequence returns if the installation's
// group sequence number matches the sequence number of the merged group config
// or not.
func (i *Installation) InstallationSequenceMatchesMergedGroupSequence() bool {
	if !i.configMergedWithGroup {
		return true
	}
	if i.GroupSequence == nil {
		return false
	}

	return i.configMergeGroupSequence == *i.GroupSequence
}

// SyncGroupAndInstallationSequence updates the installation GroupSequence value
// to reflect the hidden group Sequence value from the time the configuration
// was origianlly merged.
func (i *Installation) SyncGroupAndInstallationSequence() {
	i.GroupSequence = &i.configMergeGroupSequence
}

// MergeWithGroup merges an installation's configuration with that of a group.
// An option can be provided to include a group override summary to the
// installation.
func (i *Installation) MergeWithGroup(group *Group, includeOverrides bool) {
	if i.ConfigMergedWithGroup() {
		return
	}
	if group == nil {
		return
	}

	i.configMergedWithGroup = true
	i.configMergeGroupSequence = group.Sequence

	i.GroupOverrides = make(map[string]string)
	if group.MattermostEnv != nil && i.MattermostEnv == nil {
		i.MattermostEnv = make(EnvVarMap)
	}

	if len(group.Version) != 0 && i.Version != group.Version {
		if includeOverrides {
			i.GroupOverrides["Installation Version"] = i.Version
			i.GroupOverrides["Group Version"] = group.Version
		}
		i.Version = group.Version
	}
	if len(group.Image) != 0 && i.Image != group.Image {
		if includeOverrides {
			i.GroupOverrides["Installation Image"] = i.Image
			i.GroupOverrides["Group Image"] = group.Image
		}
		i.Image = group.Image
	}
	for key, value := range group.MattermostEnv {
		if includeOverrides {
			if _, ok := i.MattermostEnv[key]; ok {
				i.GroupOverrides[fmt.Sprintf("Installation MattermostEnv[%s]", key)] = i.MattermostEnv[key].Value
				i.GroupOverrides[fmt.Sprintf("Group MattermostEnv[%s]", key)] = value.Value
			}
		}
		i.MattermostEnv[key] = value
	}
}

// InstallationFromReader decodes a json-encoded installation from the given io.Reader.
func InstallationFromReader(reader io.Reader) (*Installation, error) {
	installation := Installation{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&installation)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &installation, nil
}

// InstallationsFromReader decodes a json-encoded list of installations from the given io.Reader.
func InstallationsFromReader(reader io.Reader) ([]*Installation, error) {
	installations := []*Installation{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&installations)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return installations, nil
}

// InstallationsCountFromReader decodes a json-encoded installations count data from the
// given io.Reader
func InstallationsCountFromReader(reader io.Reader) (int64, error) {
	installationsCount := InstallationsCount{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&installationsCount)
	if err != nil && err != io.EOF {
		return 0, err
	}

	return installationsCount.Count, nil
}
