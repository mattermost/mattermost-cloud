package model

import (
	"encoding/json"
	"fmt"
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
	utilityMetadata     *utilityMetadata
}

type utilityMetadata struct {
	Versions map[string]string
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

// UpdateUtilityMetadata takes a map of string to string representing
// any metadata related to the utility group and stores it as a []byte
// in Cluster so that it can be inserted into the database
func (c *Cluster) UpdateUtilityMetadata(versions map[string]string) error {
	var metadata []byte
	metadata, err := json.Marshal(versions)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal provided utility metadata map %#v", versions)
	}

	c.UtilityMetadata = metadata
	return nil
}

type utilityVersions struct {
	Prometheus string
	Nginx      string
	Fluentbit  string
}

// GetUtilityVersion fetches the desired version of a utility from the
// Cluster object
func (c *Cluster) GetUtilityVersion(utility string) (string, error) {

	if c.utilityMetadata == nil { // if the data doesn't exist, deserialize and cache it
		c.unmarshalAndCacheUtilityMetadata()
	}

	version, ok := c.utilityMetadata.Versions[utility]
	if !ok {
		return "", errors.New(fmt.Sprintf("couldn't get version for utility %s", utility))
	}
	return version, nil
}

func (c *Cluster) unmarshalAndCacheUtilityMetadata() error {
	output := &utilityVersions{}
	err := json.Unmarshal(c.UtilityMetadata, output)
	if err != nil {
		return errors.Wrap(err, "couldn't unmarshal stored utility metadata json")
	}

	if c.utilityMetadata == nil {
		c.utilityMetadata = &utilityMetadata{Versions: make(map[string]string)}
	}

	for _, utility := range [3]string{"prometheus", "fluentbit", "nginx"} {
		switch utility {
		case "prometheus":
			c.utilityMetadata.Versions[utility] = output.Prometheus
		case "nginx":
			c.utilityMetadata.Versions[utility] = output.Nginx
		case "fluentbit":
			c.utilityMetadata.Versions[utility] = output.Fluentbit
		}
	}

	return nil
}
