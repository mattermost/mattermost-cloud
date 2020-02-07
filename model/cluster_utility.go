package model

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

const (
	// PrometheusCanonicalName is the canonical string representation of prometheus
	PrometheusCanonicalName = "prometheus"
	// NginxCanonicalName is the canonical string representation of nginx
	NginxCanonicalName = "nginx"
	// FluentbitCanonicalName is the canonical string representation of fluentbit
	FluentbitCanonicalName = "fluentbit"
)

const (
	// PrometheusDefaultVersion defines the default version for the Helm chart
	PrometheusDefaultVersion = "10.4.0"
	// NginxDefaultVersion defines the default version for the Helm chart
	NginxDefaultVersion = "1.30.0"
	// FluentbitDefaultVersion defines the default version for the Helm chart
	FluentbitDefaultVersion = "2.8.7"
)

// UtilityMetadata is a container struct for any metadata related to
// cluster utilities that needs to be persisted in the database
type UtilityMetadata struct {
	DesiredVersions utilityVersions `json:"desiredVersions"`
	ActualVersions  utilityVersions `json:"actualVersions"`
}

type utilityVersions struct {
	Prometheus string
	Nginx      string
	Fluentbit  string
}

// SetUtilityActualVersion stores the provided version for the
// provided utility in the UtilityMetadata JSON []byte in this Cluster
func (c *Cluster) SetUtilityActualVersion(utility string, version string) error {
	oldMetadata := &UtilityMetadata{}
	if len(c.UtilityMetadata) != 0 {
		err := json.Unmarshal(c.UtilityMetadata, oldMetadata)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal existing utility metadata")
		}
	}

	setUtilityVersion(&oldMetadata.ActualVersions, utility, version)

	// reserialize and write it back to the object
	var utilityMetadata []byte
	utilityMetadata, err := json.Marshal(oldMetadata)
	if err != nil {
		return errors.Wrapf(err, "failed to store actual version info for %s", utility)
	}

	c.UtilityMetadata = utilityMetadata
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

	oldMetadata := &UtilityMetadata{}
	if len(c.UtilityMetadata) != 0 {
		// if existing data is present, unmarshal it
		err := json.Unmarshal(c.UtilityMetadata, oldMetadata)
		if err != nil {
			return errors.Wrap(err, "failed to unmarshal existing utility metadata")
		}
	}

	// assign new desired versions to the object
	for utility, version := range versions {
		setUtilityVersion(&oldMetadata.DesiredVersions, utility, version)
	}

	// reserialize and write it back to the object
	var utilityMetadata []byte
	utilityMetadata, err := json.Marshal(oldMetadata)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal provided utility metadata map %#v", versions)
	}

	c.UtilityMetadata = utilityMetadata
	return nil
}

// DesiredUtilityVersion fetches the desired version of a utility from the
// Cluster object
func (c *Cluster) DesiredUtilityVersion(utility string) (string, error) {
	// some clusters may only be using pinned stable version, so an
	// empty UtilityMetadata field is possible; in this context it means
	// "utility"'s desired version is nothing
	if len(c.UtilityMetadata) == 0 {
		return "", nil
	}

	output := &UtilityMetadata{}
	err := json.Unmarshal(c.UtilityMetadata, output)
	if err != nil {
		return "", errors.Wrap(err, "couldn't unmarshal stored utility metadata json")
	}

	return getUtilityVersion(&output.DesiredVersions, utility), nil
}

// ActualUtilityVersion fetches the desired version of a utility from the
// Cluster object
func (c *Cluster) ActualUtilityVersion(utility string) (string, error) {
	output := &UtilityMetadata{}
	err := json.Unmarshal(c.UtilityMetadata, output)
	if err != nil {
		return "", errors.Wrap(err, "couldn't unmarshal stored utility metadata json")
	}

	return getUtilityVersion(&output.ActualVersions, utility), nil
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
	}
}
