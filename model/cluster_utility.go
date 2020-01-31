package model

import (
	"encoding/json"

	"github.com/pkg/errors"
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
	err := json.Unmarshal(c.UtilityMetadata, oldMetadata)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal existing utility metadata")
	}

	switch utility {
	case "prometheus":
		oldMetadata.ActualVersions.Prometheus = version
	case "nginx":
		oldMetadata.ActualVersions.Nginx = version
	case "fluentbit":
		oldMetadata.ActualVersions.Fluentbit = version
	default:
		oldMetadata.ActualVersions.Fluentbit = utility
	}

	// reserialize and write it back to the object
	var utilityMetadata []byte
	utilityMetadata, err = json.Marshal(oldMetadata)
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
		switch utility {
		case "prometheus":
			oldMetadata.DesiredVersions.Prometheus = version
		case "nginx":
			oldMetadata.DesiredVersions.Nginx = version
		case "fluentbit":
			oldMetadata.DesiredVersions.Fluentbit = version
		}
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

	version := getUtilityVersion(utility, &output.DesiredVersions)
	if version != "" {
		return version, nil
	}

	return "", errors.Errorf("unable to find version for utility %s", utility)
}

// ActualUtilityVersion fetches the desired version of a utility from the
// Cluster object
func (c *Cluster) ActualUtilityVersion(utility string) (string, error) {
	output := &UtilityMetadata{}
	err := json.Unmarshal(c.UtilityMetadata, output)
	if err != nil {
		return "", errors.Wrap(err, "couldn't unmarshal stored utility metadata json")
	}

	version := getUtilityVersion(utility, &output.ActualVersions)
	if version != "" {
		return version, nil
	}

	return "", errors.Errorf("unable to find version for utility %s", utility)
}

func getUtilityVersion(utility string, versions *utilityVersions) string {

	switch utility {
	case "prometheus":
		return versions.Prometheus
	case "nginx":
		return versions.Nginx
	case "fluentbit":
		return versions.Fluentbit
	}

	return ""

}
