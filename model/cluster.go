// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
	"regexp"
)

//go:generate provisioner-code-gen generate --out-file=cluster_gen.go --boilerplate-file=../hack/boilerplate/boilerplate.generatego.txt --type=github.com/mattermost/mattermost-cloud/model.Cluster --generator=get_id,get_state,is_deleted,as_resources

// Cluster represents a Kubernetes cluster.
type Cluster struct {
	ID                          string
	Name                        string
	State                       string
	Provider                    string
	ProviderMetadataAWS         *AWSMetadata              `json:"ProviderMetadataAWS,omitempty"`
	ProviderMetadataExternal    *ExternalProviderMetadata `json:"ProviderMetadataExternal,omitempty"`
	Provisioner                 string
	ProvisionerMetadataKops     *KopsMetadata            `json:"ProvisionerMetadataKops,omitempty"`
	ProvisionerMetadataEKS      *EKSMetadata             `json:"ProvisionerMetadataEKS,omitempty"`
	ProvisionerMetadataExternal *ExternalClusterMetadata `json:"ProvisionerMetadataExternal,omitempty"`
	Networking                  string
	UtilityMetadata             *UtilityMetadata
	PgBouncerConfig             *PgBouncerConfig
	AllowInstallations          bool
	CreateAt                    int64
	DeleteAt                    int64
	APISecurityLock             bool
	SchedulingLockAcquiredBy    *string
	SchedulingLockAcquiredAt    int64
	LockAcquiredBy              *string
	LockAcquiredAt              int64
}

// Clone returns a deep copy the cluster.
func (c *Cluster) Clone() *Cluster {
	var clone Cluster
	data, _ := json.Marshal(c)
	json.Unmarshal(data, &clone)

	return &clone
}

// ToDTO expands cluster to ClusterDTO.
func (c *Cluster) ToDTO(annotations []*Annotation) *ClusterDTO {
	return &ClusterDTO{
		Cluster:     c,
		Annotations: annotations,
	}
}

// VpcID is a safe way to return the VPC ID of the cluster if it has one
// regardless of the cluster metadata type.
func (c *Cluster) VpcID() string {
	var vpcID string
	switch c.Provisioner {
	case ProvisionerKops:
		vpcID = c.ProvisionerMetadataKops.VPC
	case ProvisionerEKS:
		vpcID = c.ProvisionerMetadataEKS.VPC
	case ProvisionerExternal:
		vpcID = c.ProvisionerMetadataExternal.VPC
	}

	return vpcID
}

// HasAWSInfrastructure returns if a cluster is in a standard VPC with our
// managed database and filestore resources.
func (c *Cluster) HasAWSInfrastructure() bool {
	if c.Provider == ProviderExternal {
		return c.ProviderMetadataExternal.HasAWSInfrastructure
	}

	return true
}

func (c *Cluster) ApplyClusterUpdatePatch(patchRequest *UpdateClusterRequest) bool {
	var applied bool
	if patchRequest.Name != nil && *patchRequest.Name != c.Name {
		applied = true
		c.Name = *patchRequest.Name
	}
	if patchRequest.AllowInstallations != nil && *patchRequest.AllowInstallations != c.AllowInstallations {
		applied = true
		c.AllowInstallations = *patchRequest.AllowInstallations
	}

	return applied
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
	Paging
	Annotations *AnnotationsFilter
}

// AnnotationsFilter describes filter based on Annotations.
type AnnotationsFilter struct {
	// MatchAllIDs contains all Annotation IDs which need to be set on a Cluster for it to be included in the result.
	MatchAllIDs []string
}

// EKS only support x.xx versioning
var clusterVersionMatcher = regexp.MustCompile(`^(([0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3})|([0-9]{1,3}.[0-9]{1,3})|(latest))$`)

// ValidClusterVersion returns true if the provided version is either "latest"
// or a valid k8s version number.
func ValidClusterVersion(name string) bool {
	return clusterVersionMatcher.MatchString(name)
}

// IsValidKMSARN checks if a string is a valid KMS ARN.
func IsValidKMSARN(s string) bool {
	// Define a regular expression pattern for a KMS ARN
	// Modify this pattern if needed to match your specific requirements.
	pattern := `^arn:aws:kms:[a-zA-Z0-9_-]+:[0-9]+:key/[a-f0-9-]+$`

	// Compile the regular expression
	re := regexp.MustCompile(pattern)

	// Use the regular expression to match the input string
	return re.MatchString(s)
}
