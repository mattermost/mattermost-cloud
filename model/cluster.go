package model

import (
	"encoding/json"
	"io"
	"regexp"
)

// Cluster represents a Kubernetes cluster.
type Cluster struct {
	ID                      string
	State                   string
	Provider                string
	ProviderMetadataAWS     *AWSMetadata
	Provisioner             string
	ProvisionerMetadataKops *KopsMetadata
	UtilityMetadata         *UtilityMetadata
	AllowInstallations      bool
	Version                 string
	Size                    string
	CreateAt                int64
	DeleteAt                int64
	LockAcquiredBy          *string
	LockAcquiredAt          int64
}

// Clone returns a deep copy the cluster.
func (c *Cluster) Clone() *Cluster {
	var clone Cluster
	data, _ := json.Marshal(c)
	json.Unmarshal(data, &clone)

	return &clone
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
