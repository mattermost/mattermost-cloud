// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/pkg/errors"
)

const (
	// PrometheusOperatorCanonicalName is the canonical string representation of prometheus operator
	PrometheusOperatorCanonicalName = "prometheus-operator"
	// ThanosCanonicalName is the canonical string representation of thanos
	ThanosCanonicalName = "thanos"
	// NginxCanonicalName is the canonical string representation of nginx
	NginxCanonicalName = "nginx"
	// FluentbitCanonicalName is the canonical string representation of fluentbit
	FluentbitCanonicalName = "fluentbit"
	// TeleportCanonicalName is the canonical string representation of teleport
	TeleportCanonicalName = "teleport"
)

var (
	// PrometheusOperatorDefaultVersion defines the default version for the Helm chart
	PrometheusOperatorDefaultVersion = &HelmUtilityVersion{Chart: "9.4.4", ValuesPath: "production"}
	// ThanosDefaultVersion defines the default version for the Helm chart
	ThanosDefaultVersion = &HelmUtilityVersion{Chart: "2.4.3", ValuesPath: "production"}
	// NginxDefaultVersion defines the default version for the Helm chart
	NginxDefaultVersion = &HelmUtilityVersion{Chart: "2.15.0", ValuesPath: "production"}
	// FluentbitDefaultVersion defines the default version for the Helm chart
	FluentbitDefaultVersion = &HelmUtilityVersion{Chart: "2.8.7", ValuesPath: "production"}
	// TeleportDefaultVersion defines the default version for the Helm chart
	TeleportDefaultVersion = &HelmUtilityVersion{Chart: "0.3.0", ValuesPath: "production"}
)

// UtilityVersion is an interface that provides the necessary methods
// to discover the numerical version of the Utility as well as an
// identifier necessary in order to fetch any other configuration
type UtilityVersion interface {
	Version() string
	SetVersion(version string)

	Values() string
	SetValues(valuesLocation string)
}

// UnmarshalJSON is a custom JSON unmarshaler that can handle both the
// old Version string type and the new type. It is entirely
// self-contained, including types, so that it can be easily removed
// when no more clusters exist with the old version format.
// TODO DELETE THIS
func (h *UtilityGroupVersions) UnmarshalJSON(bytes []byte) error {
	type utilityGroupVersions struct {
		PrometheusOperator *HelmUtilityVersion
		Thanos             *HelmUtilityVersion
		Nginx              *HelmUtilityVersion
		Fluentbit          *HelmUtilityVersion
		Teleport           *HelmUtilityVersion
	}
	type oldUtilityGroupVersions struct {
		PrometheusOperator string
		Thanos             string
		Nginx              string
		Fluentbit          string
		Teleport           string
	}

	var utilGrpVers *utilityGroupVersions = &utilityGroupVersions{}
	var oldUtilGrpVers *oldUtilityGroupVersions = &oldUtilityGroupVersions{}
	err := json.Unmarshal(bytes, utilGrpVers)
	if err != nil {
		secondErr := json.Unmarshal(bytes, oldUtilGrpVers)
		if secondErr != nil {
			return fmt.Errorf("%s and %s", errors.Wrap(err, "failed to unmarshal to new HelmUtilityVersion"), errors.Wrap(secondErr, "failed to unmarshal to old HelmUtilityVersion type"))
		}

		h.PrometheusOperator = &HelmUtilityVersion{Chart: oldUtilGrpVers.PrometheusOperator}
		h.Thanos = &HelmUtilityVersion{Chart: oldUtilGrpVers.Thanos}
		h.Nginx = &HelmUtilityVersion{Chart: oldUtilGrpVers.Nginx}
		h.Fluentbit = &HelmUtilityVersion{Chart: oldUtilGrpVers.Fluentbit}
		h.Teleport = &HelmUtilityVersion{Chart: oldUtilGrpVers.Teleport}
		return nil
	}

	h.PrometheusOperator = utilGrpVers.PrometheusOperator
	h.Thanos = utilGrpVers.Thanos
	h.Nginx = utilGrpVers.Nginx
	h.Fluentbit = utilGrpVers.Fluentbit
	h.Teleport = utilGrpVers.Teleport
	return nil
}

// UtilityGroupVersions holds the concrete metadata for any cluster
// utilities
type UtilityGroupVersions struct {
	PrometheusOperator *HelmUtilityVersion
	Thanos             *HelmUtilityVersion
	Nginx              *HelmUtilityVersion
	Fluentbit          *HelmUtilityVersion
	Teleport           *HelmUtilityVersion
}

// UtilityMetadata is a container struct for any metadata related to
// cluster utilities that needs to be persisted in the database
type UtilityMetadata struct {
	DesiredVersions UtilityGroupVersions
	ActualVersions  UtilityGroupVersions
}

// NewUtilityMetadata creates an instance of UtilityMetadata given the raw
// utility metadata.
func NewUtilityMetadata(metadataBytes []byte) (*UtilityMetadata, error) {
	// Check if length of metadata is 0 as opposed to if the value is nil. This
	// is done to avoid an issue encountered where the metadata value provided
	// had a length of 0, but had non-zero capacity.
	if len(metadataBytes) == 0 || string(metadataBytes) == "null" {
		// TODO: remove "null" check after sqlite is gone.
		return nil, nil
	}

	utilityMetadata := UtilityMetadata{}
	err := json.Unmarshal(metadataBytes, &utilityMetadata)
	if err != nil {
		return nil, err
	}

	return &utilityMetadata, nil
}

// SetUtilityActualVersion stores the provided version for the
// provided utility in the UtilityMetadata JSON []byte in this Cluster
func (c *Cluster) SetUtilityActualVersion(utility string, version UtilityVersion) error {
	metadata := &UtilityMetadata{}
	if c.UtilityMetadata != nil {
		metadata = c.UtilityMetadata
	}

	setUtilityVersion(&metadata.ActualVersions, utility, version)

	c.UtilityMetadata = metadata
	return nil
}

// SetUtilityDesiredVersions takes a map of string to string representing
// any metadata related to the utility group and stores it as a []byte
// in Cluster so that it can be inserted into the database
func (c *Cluster) SetUtilityDesiredVersions(versions map[string]UtilityVersion) error {
	// If a version is originally not provided, we want to install the
	// "stable" version. However, if a version is specified, the user
	// might later want to move the version back to tracking the stable
	// release.
	for utility, version := range versions {
		if version == nil {
			versions[utility] = nil
		}
	}

	metadata := &UtilityMetadata{}
	if c.UtilityMetadata != nil {
		metadata = c.UtilityMetadata
	}

	// assign new desired versions to the object
	for utility, version := range versions {
		setUtilityVersion(&metadata.DesiredVersions, utility, version)
	}

	c.UtilityMetadata = metadata
	return nil
}

// DesiredUtilityVersion fetches the desired version of a utility from the
// Cluster object
func (c *Cluster) DesiredUtilityVersion(utility string) UtilityVersion {
	// some clusters may only be using pinned stable version, so an
	// empty UtilityMetadata field is possible; in this context it means
	// "utility"'s desired version is nothing
	if c.UtilityMetadata == nil {
		return nil
	}

	return getUtilityVersion(c.UtilityMetadata.DesiredVersions, utility)
}

// ActualUtilityVersion fetches the desired version of a utility from the
// Cluster object
func (c *Cluster) ActualUtilityVersion(utility string) UtilityVersion {
	return getUtilityVersion(c.UtilityMetadata.ActualVersions, utility)
}

// UtilityMetadataFromReader produces a UtilityMetadata object from
// the JSON representation embedded in a io.Reader
func UtilityMetadataFromReader(reader io.Reader) (*UtilityMetadata, error) {
	utilityMetadata := UtilityMetadata{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&utilityMetadata)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &utilityMetadata, nil
}

// Gets the version for a utility from a utilityVersions struct using
// the utility's name's string representation for lookup
func getUtilityVersion(versions UtilityGroupVersions, utility string) UtilityVersion {
	switch utility {
	case PrometheusOperatorCanonicalName:
		return versions.PrometheusOperator
	case ThanosCanonicalName:
		return versions.Thanos
	case NginxCanonicalName:
		return versions.Nginx
	case FluentbitCanonicalName:
		return versions.Fluentbit
	case TeleportCanonicalName:
		return versions.Teleport
	}

	return nil
}

// setUtilityVersion will assign the version in desiredVersion to the
// utility whose name's string representation matches one of the known
// utilities with a version field in utilityVersion struct in the
// first argument
func setUtilityVersion(versions *UtilityGroupVersions, utility string, desiredVersion UtilityVersion) {
	if desiredVersion == nil {
		return
	}

	switch utility {
	case PrometheusOperatorCanonicalName:
		versions.PrometheusOperator = desiredVersion.(*HelmUtilityVersion)
	case ThanosCanonicalName:
		versions.Thanos = desiredVersion.(*HelmUtilityVersion)
	case NginxCanonicalName:
		versions.Nginx = desiredVersion.(*HelmUtilityVersion)
	case FluentbitCanonicalName:
		versions.Fluentbit = desiredVersion.(*HelmUtilityVersion)
	case TeleportCanonicalName:
		versions.Teleport = desiredVersion.(*HelmUtilityVersion)
	}
}

// HelmUtilityVersion holds the chart version and the version of the
// values file
type HelmUtilityVersion struct {
	Chart      string
	ValuesPath string
}

// Version returns the Helm chart version
func (u *HelmUtilityVersion) Version() string {
	return u.Chart
}

// Values returns the name of the branch on which to find the correct
// values file
func (u *HelmUtilityVersion) Values() string {
	return u.ValuesPath
}

// SetValues sets the ValuesPath and satisfies the UtilityVersion interface
func (u *HelmUtilityVersion) SetValues(valuesLocation string) {
	u.ValuesPath = valuesLocation
}

// SetVersion sets the chart version and satisfies the UtilityVersion interface
func (u *HelmUtilityVersion) SetVersion(version string) {
	u.Chart = version
}
