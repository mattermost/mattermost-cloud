package model

import (
	"encoding/json"
	"io"
	"regexp"

	"github.com/pkg/errors"
)

// Cluster represents a Kubernetes cluster.
type Cluster struct {
	ID                  string
	Provider            string
	Provisioner         string
	ProviderMetadata    []byte `json:",omitempty"`
	ProvisionerMetadata []byte `json:",omitempty"`
	AllowInstallations  bool
	Version             string
	Size                string
	State               string
	CreateAt            int64
	DeleteAt            int64
	LockAcquiredBy      *string
	LockAcquiredAt      int64
	UtilityMetadata     []byte `json:",omitempty"`
}

// Clone returns a deep copy the cluster.
func (c *Cluster) Clone() *Cluster {
	var clone Cluster
	data, _ := json.Marshal(c)
	json.Unmarshal(data, &clone)

	return &clone
}

// SetProviderMetadata is a helper method to encode an interface{} as the corresponding bytes.
func (c *Cluster) SetProviderMetadata(data interface{}) error {
	if data == nil {
		c.ProviderMetadata = nil
		return nil
	}

	providerMetadata, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to set provider metadata")
	}

	c.ProviderMetadata = providerMetadata
	return nil
}

// SetProvisionerMetadata is a helper method to encode an interface{} as the corresponding bytes.
func (c *Cluster) SetProvisionerMetadata(data interface{}) error {
	if data == nil {
		c.ProvisionerMetadata = nil
		return nil
	}

	provisionerMetadata, err := json.Marshal(data)
	if err != nil {
		return errors.Wrap(err, "failed to set provisioner metadata")
	}

	c.ProvisionerMetadata = provisionerMetadata
	return nil
}

// ClusterFromReader decodes a json-encoded cluster from the given io.Reader.
func ClusterFromReader(reader io.Reader) (*Cluster, error) {
	cluster := Cluster{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&cluster)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &cluster, nil
}

// ClustersFromReader decodes a json-encoded list of clusters from the given io.Reader.
func ClustersFromReader(reader io.Reader) ([]*Cluster, error) {
	clusters := []*Cluster{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&clusters)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return clusters, nil
}

// ClusterFilter describes the parameters used to constrain a set of clusters.
type ClusterFilter struct {
	Page           int
	PerPage        int
	IncludeDeleted bool
}

var clusterVersionMatcher = regexp.MustCompile(`^(([0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3})|(latest))$`)

// ValidClusterVersion returns true if the provided version is either "latest"
// or a valid k8s version number.
func ValidClusterVersion(name string) bool {
	return clusterVersionMatcher.MatchString(name)
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

type UtilityMetadata struct {
	DesiredVersions utilityVersions `json:"desiredVersions"`
	ActualVersions  utilityVersions `json:"actualVersions"`
}

type utilityVersions struct {
	Prometheus string
	Nginx      string
	Fluentbit  string
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

	switch utility {
	case "prometheus":
		return output.DesiredVersions.Prometheus, nil
	case "nginx":
		return output.DesiredVersions.Nginx, nil
	case "fluentbit":
		return output.DesiredVersions.Fluentbit, nil
	}

	return "", errors.Errorf("unable to find version for utility %s", utility)
}

// DesiredUtilityVersion fetches the desired version of a utility from the
// Cluster object
func (c *Cluster) ActualUtilityVersion(utility string) (string, error) {
	output := &UtilityMetadata{}
	err := json.Unmarshal(c.UtilityMetadata, output)
	if err != nil {
		return "", errors.Wrap(err, "couldn't unmarshal stored utility metadata json")
	}

	switch utility {
	case "prometheus":
		return output.ActualVersions.Prometheus, nil
	case "nginx":
		return output.ActualVersions.Nginx, nil
	case "fluentbit":
		return output.ActualVersions.Fluentbit, nil
	}

	return "", errors.Errorf("unable to find version for utility %s", utility)
}
