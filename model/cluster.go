// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
	"regexp"
)

const (
	// MattermostWebhook is the name of the Environment Variable which
	// may contain a Mattermost webhook to send notifications to a Mattermost installation
	MattermostWebhook = "mattermost-webhook"
	// MattermostChannel is the name of the Environment Variable which
	// may contain a Mattermost channel in which notifications are going to be sent
	MattermostChannel = "mattermost-channel"
	// KubecostToken is the name of the Environment Variable which
	// may contain a Kubecost token which kubecost helm chart needs
	KubecostToken = "kubecost-token"
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
	CreateAt                int64
	DeleteAt                int64
	APISecurityLock         bool
	LockAcquiredBy          *string
	LockAcquiredAt          int64
	Networking              string
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

var clusterVersionMatcher = regexp.MustCompile(`^(([0-9]{1,3}.[0-9]{1,3}.[0-9]{1,3})|(latest))$`)

// ValidClusterVersion returns true if the provided version is either "latest"
// or a valid k8s version number.
func ValidClusterVersion(name string) bool {
	return clusterVersionMatcher.MatchString(name)
}
