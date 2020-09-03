// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
)

const (
	// PrometheusCanonicalName is the canonical string representation of prometheus
	PrometheusCanonicalName = "prometheus"
	// NginxCanonicalName is the canonical string representation of nginx
	NginxCanonicalName = "nginx"
	// FluentbitCanonicalName is the canonical string representation of fluentbit
	FluentbitCanonicalName = "fluentbit"
	// TeleportCanonicalName is the canonical string representation of teleport
	TeleportCanonicalName = "teleport"
)

const (
	// PrometheusDefaultVersion defines the default version for the Helm chart
	PrometheusDefaultVersion = "10.4.0"
	// NginxDefaultVersion defines the default version for the Helm chart
	NginxDefaultVersion = "2.15.0"
	// FluentbitDefaultVersion defines the default version for the Helm chart
	FluentbitDefaultVersion = "2.8.7"
	// TeleportDefaultVersion defines the default version for the Helm chart
	TeleportDefaultVersion = "0.3.0"
)

// UtilityMetadata is a container struct for any metadata related to
// cluster utilities that needs to be persisted in the database
type UtilityMetadata struct {
	DesiredVersions utilityVersions
	ActualVersions  utilityVersions
}

type utilityVersions struct {
	Prometheus string
	Nginx      string
	Fluentbit  string
	Teleport   string
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
func (c *Cluster) SetUtilityActualVersion(utility string, version string) error {
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
func (c *Cluster) SetUtilityDesiredVersions(versions map[string]string) error {
	// If a version is originally not provided, we want to install the
	// "stable" version. However, if a version is specified, the user
	// might later want to move the version back to tracking the stable
	// release.
	for utility, version := range versions {
		if version == "stable" {
			versions[utility] = ""
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
func (c *Cluster) DesiredUtilityVersion(utility string) (string, error) {
	// some clusters may only be using pinned stable version, so an
	// empty UtilityMetadata field is possible; in this context it means
	// "utility"'s desired version is nothing
	if c.UtilityMetadata == nil {
		return "", nil
	}

	return getUtilityVersion(&c.UtilityMetadata.DesiredVersions, utility), nil
}

// ActualUtilityVersion fetches the desired version of a utility from the
// Cluster object
func (c *Cluster) ActualUtilityVersion(utility string) (string, error) {
	return getUtilityVersion(&c.UtilityMetadata.ActualVersions, utility), nil
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
func getUtilityVersion(versions *utilityVersions, utility string) string {
	switch utility {
	case PrometheusCanonicalName:
		return versions.Prometheus
	case NginxCanonicalName:
		return versions.Nginx
	case FluentbitCanonicalName:
		return versions.Fluentbit
	case TeleportCanonicalName:
		return versions.Teleport
	}

	return ""
}

// setUtilityVersion will assign the version in desiredVersion to the
// utility whose name's string representation matches one of the known
// utilities with a version field in utilityVersion struct in the
// first argument
func setUtilityVersion(versions *utilityVersions, utility, desiredVersion string) {
	switch utility {
	case PrometheusCanonicalName:
		versions.Prometheus = desiredVersion
	case NginxCanonicalName:
		versions.Nginx = desiredVersion
	case FluentbitCanonicalName:
		versions.Fluentbit = desiredVersion
	case TeleportCanonicalName:
		versions.Teleport = desiredVersion
	}
}
