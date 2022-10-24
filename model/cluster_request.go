// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
	"net/url"

	"github.com/pkg/errors"
)

const (
	// NetworkingCalico is Calico networking plugin.
	NetworkingCalico = "calico"
	// NetworkingAmazon is Amazon networking plugin.
	NetworkingAmazon = "amazon-vpc-routed-eni"
)

var (
	defaultEKSRoleARN       string
	defaultNodeGroupRoleARN string
)

// CreateClusterRequest specifies the parameters for a new cluster.
type CreateClusterRequest struct {
	Provider               string                         `json:"provider,omitempty"`
	Zones                  []string                       `json:"zones,omitempty"`
	Version                string                         `json:"version,omitempty"`
	KopsAMI                string                         `json:"kops-ami,omitempty"`
	MasterInstanceType     string                         `json:"master-instance-type,omitempty"`
	MasterCount            int64                          `json:"master-count,omitempty"`
	NodeInstanceType       string                         `json:"node-instance-type,omitempty"`
	NodeMinCount           int64                          `json:"node-min-count,omitempty"`
	NodeMaxCount           int64                          `json:"node-max-count,omitempty"`
	AllowInstallations     bool                           `json:"allow-installations,omitempty"`
	APISecurityLock        bool                           `json:"api-security-lock,omitempty"`
	DesiredUtilityVersions map[string]*HelmUtilityVersion `json:"utility-versions,omitempty"`
	Annotations            []string                       `json:"annotations,omitempty"`
	Networking             string                         `json:"networking,omitempty"`
	VPC                    string                         `json:"vpc,omitempty"`
	MaxPodsPerNode         int64
	EKSConfig              *EKSConfig `json:"EKSConfig,omitempty"`
}

// EKSConfig is EKS cluster configuration.
type EKSConfig struct {
	ClusterRoleARN *string                 `json:"clusterRoleARN,omitempty"`
	NodeGroups     map[string]EKSNodeGroup `json:"nodeGroups,omitempty"`
}

func (request *CreateClusterRequest) setUtilityDefaults(utilityName string) {
	reqDesiredUtilityVersion, ok := request.DesiredUtilityVersions[utilityName]
	if !ok {
		request.DesiredUtilityVersions[utilityName] = DefaultUtilityVersions[utilityName]
		return
	}
	if reqDesiredUtilityVersion.Chart == "" {
		reqDesiredUtilityVersion.Chart = DefaultUtilityVersions[utilityName].Chart
	}
	if reqDesiredUtilityVersion.ValuesPath == "" {
		reqDesiredUtilityVersion.ValuesPath = DefaultUtilityVersions[utilityName].ValuesPath
	}
}

func (request *CreateClusterRequest) setUtilitiesDefaults() {
	for utilityName := range DefaultUtilityVersions {
		request.setUtilityDefaults(utilityName)
	}
}

// SetDefaults sets the default values for a cluster create request.
func (request *CreateClusterRequest) SetDefaults() {
	if len(request.Provider) == 0 {
		request.Provider = ProviderAWS
	}
	if len(request.Version) == 0 {
		request.Version = "latest"
	}
	if len(request.Zones) == 0 {
		request.Zones = []string{"us-east-1a"}
	}
	if len(request.MasterInstanceType) == 0 {
		request.MasterInstanceType = "t3.medium"
	}
	if request.MasterCount == 0 {
		request.MasterCount = 1
	}
	if len(request.NodeInstanceType) == 0 {
		request.NodeInstanceType = "m5.large"
	}
	if request.NodeMinCount == 0 {
		request.NodeMinCount = 2
	}
	if request.NodeMaxCount == 0 {
		request.NodeMaxCount = request.NodeMinCount
	}
	if request.MaxPodsPerNode == 0 {
		request.MaxPodsPerNode = 200
	}
	if len(request.Networking) == 0 {
		request.Networking = NetworkingCalico
	}
	if request.EKSConfig != nil {
		if request.EKSConfig.ClusterRoleARN == nil {
			request.EKSConfig.ClusterRoleARN = &defaultEKSRoleARN
		}

		for _, ng := range request.EKSConfig.NodeGroups {
			if ng.RoleARN == nil {
				ng.RoleARN = &defaultNodeGroupRoleARN
			}
		}
	}

	if request.DesiredUtilityVersions == nil {
		request.DesiredUtilityVersions = make(map[string]*HelmUtilityVersion)
	}
	request.setUtilitiesDefaults()
}

// Validate validates the values of a cluster create request.
func (request *CreateClusterRequest) Validate() error {
	if request.Provider != ProviderAWS {
		return errors.Errorf("unsupported provider %s", request.Provider)
	}
	if !ValidClusterVersion(request.Version) {
		return errors.Errorf("unsupported cluster version %s", request.Version)
	}
	if request.MasterCount < 1 {
		return errors.Errorf("master count (%d) must be 1 or greater", request.MasterCount)
	}
	if request.NodeMinCount < 1 {
		return errors.Errorf("node min count (%d) must be 1 or greater", request.NodeMinCount)
	}
	if request.NodeMaxCount != request.NodeMinCount {
		return errors.Errorf("node min (%d) and max (%d) counts must match", request.NodeMinCount, request.NodeMaxCount)
	}
	if request.MaxPodsPerNode < 10 {
		return errors.Errorf("max pods per node (%d) must be 10 or greater", request.MaxPodsPerNode)
	}
	// TODO: check zones and instance types?

	if request.EKSConfig != nil {
		if request.EKSConfig.ClusterRoleARN == nil || *request.EKSConfig.ClusterRoleARN == "" {
			return errors.New("cluster role ARN for EKS cluster cannot be empty")
		}
		if len(request.EKSConfig.NodeGroups) == 0 {
			return errors.New("at least 1 node group is required when using EKS")
		}
	}

	if !contains(GetSupportedCniList(), request.Networking) {
		return errors.Errorf("unsupported cluster networking option %s", request.Networking)
	}
	return nil
}

// GetSupportedCniList starting with three supported CNI networking options, we can add more as required
func GetSupportedCniList() []string {
	return []string{"amazon-vpc-routed-eni", "amazonvpc", "weave", "canal", "calico"}
}

// contains checks if a string is present in a slice
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}

// NewCreateClusterRequestFromReader will create a CreateClusterRequest from an
// io.Reader with JSON data.
func NewCreateClusterRequestFromReader(reader io.Reader) (*CreateClusterRequest, error) {
	var createClusterRequest CreateClusterRequest
	err := json.NewDecoder(reader).Decode(&createClusterRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode create cluster request")
	}

	createClusterRequest.SetDefaults()
	err = createClusterRequest.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "create cluster request failed validation")
	}

	return &createClusterRequest, nil
}

// GetClustersRequest describes the parameters to request a list of clusters.
type GetClustersRequest struct {
	Paging
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetClustersRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	request.Paging.AddToQuery(q)

	u.RawQuery = q.Encode()
}

// UpdateClusterRequest specifies the parameters available for updating a cluster.
type UpdateClusterRequest struct {
	AllowInstallations bool
}

// NewUpdateClusterRequestFromReader will create an UpdateClusterRequest from an io.Reader with JSON data.
func NewUpdateClusterRequestFromReader(reader io.Reader) (*UpdateClusterRequest, error) {
	var updateClusterRequest UpdateClusterRequest
	err := json.NewDecoder(reader).Decode(&updateClusterRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode update cluster request")
	}
	return &updateClusterRequest, nil
}

// PatchUpgradeClusterRequest specifies the parameters for upgrading a cluster.
type PatchUpgradeClusterRequest struct {
	Version        *string        `json:"version,omitempty"`
	KopsAMI        *string        `json:"kops-ami,omitempty"`
	RotatorConfig  *RotatorConfig `json:"rotatorConfig,omitempty"`
	MaxPodsPerNode *int64
}

// Validate validates the values of a cluster upgrade request.
func (p *PatchUpgradeClusterRequest) Validate() error {
	if p.Version != nil && !ValidClusterVersion(*p.Version) {
		return errors.Errorf("unsupported cluster version %s", *p.Version)
	}
	if p.MaxPodsPerNode != nil && *p.MaxPodsPerNode < 10 {
		return errors.Errorf("max pods per node (%d) must be 10 or greater", *p.MaxPodsPerNode)
	}

	if p.RotatorConfig != nil {
		if err := p.RotatorConfig.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Apply applies the patch to the given cluster's metadata.
func (p *PatchUpgradeClusterRequest) Apply(metadata *KopsMetadata) bool {
	changes := &KopsMetadataRequestedState{}

	var applied bool
	if p.Version != nil && *p.Version != metadata.Version {
		applied = true
		changes.Version = *p.Version
	}
	if p.KopsAMI != nil && *p.KopsAMI != metadata.AMI {
		applied = true
		changes.AMI = *p.KopsAMI
	}
	if p.MaxPodsPerNode != nil && *p.MaxPodsPerNode != metadata.MaxPodsPerNode {
		applied = true
		changes.MaxPodsPerNode = *p.MaxPodsPerNode
	}

	if metadata.RotatorRequest == nil {
		metadata.RotatorRequest = &RotatorMetadata{}
	}

	if applied {
		metadata.ChangeRequest = changes
		metadata.RotatorRequest.Config = p.RotatorConfig
	}

	return applied
}

// NewUpgradeClusterRequestFromReader will create an UpgradeClusterRequest from an io.Reader with JSON data.
func NewUpgradeClusterRequestFromReader(reader io.Reader) (*PatchUpgradeClusterRequest, error) {
	var upgradeClusterRequest PatchUpgradeClusterRequest
	err := json.NewDecoder(reader).Decode(&upgradeClusterRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode upgrade cluster request")
	}

	err = upgradeClusterRequest.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "upgrade cluster request failed validation")
	}

	return &upgradeClusterRequest, nil
}

// PatchClusterSizeRequest specifies the parameters for resizing a cluster.
type PatchClusterSizeRequest struct {
	NodeInstanceType *string        `json:"node-instance-type,omitempty"`
	NodeMinCount     *int64         `json:"node-min-count,omitempty"`
	NodeMaxCount     *int64         `json:"node-max-count,omitempty"`
	RotatorConfig    *RotatorConfig `json:"rotatorConfig,omitempty"`
}

// Validate validates the values of a PatchClusterSizeRequest.
func (p *PatchClusterSizeRequest) Validate() error {
	if p.NodeInstanceType != nil && len(*p.NodeInstanceType) == 0 {
		return errors.New("node instance type cannot be a blank value")
	}
	if p.NodeMinCount != nil && *p.NodeMinCount < 1 {
		return errors.New("node min count has to be 1 or greater")
	}
	if p.NodeMinCount != nil && p.NodeMaxCount != nil &&
		*p.NodeMaxCount < *p.NodeMinCount {
		return errors.Errorf("node max count (%d) can't be less than min count (%d)", *p.NodeMaxCount, *p.NodeMinCount)
	}

	if p.RotatorConfig != nil {
		if err := p.RotatorConfig.Validate(); err != nil {
			return err
		}
	}

	return nil
}

// Apply applies the patch to the given cluster's kops metadata.
func (p *PatchClusterSizeRequest) Apply(metadata *KopsMetadata) bool {
	changes := &KopsMetadataRequestedState{}

	var applied bool
	if p.NodeInstanceType != nil && *p.NodeInstanceType != metadata.NodeInstanceType {
		applied = true
		changes.NodeInstanceType = *p.NodeInstanceType
	}
	if p.NodeMinCount != nil && *p.NodeMinCount != metadata.NodeMinCount {
		applied = true
		changes.NodeMinCount = *p.NodeMinCount
	}
	if p.NodeMaxCount != nil && *p.NodeMaxCount != metadata.NodeMaxCount {
		applied = true
		changes.NodeMaxCount = *p.NodeMaxCount
	}

	if metadata.RotatorRequest == nil {
		metadata.RotatorRequest = &RotatorMetadata{}
	}

	if applied {
		metadata.ChangeRequest = changes
		metadata.RotatorRequest.Config = p.RotatorConfig
	}

	return applied
}

// NewResizeClusterRequestFromReader will create an PatchClusterSizeRequest from an io.Reader with JSON data.
func NewResizeClusterRequestFromReader(reader io.Reader) (*PatchClusterSizeRequest, error) {
	var patchClusterSizeRequest PatchClusterSizeRequest
	err := json.NewDecoder(reader).Decode(&patchClusterSizeRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode resize cluster request")
	}

	err = patchClusterSizeRequest.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "resize cluster request failed validation")
	}

	return &patchClusterSizeRequest, nil
}

// ProvisionClusterRequest contains metadata related to changing the installed cluster state.
type ProvisionClusterRequest struct {
	DesiredUtilityVersions map[string]*HelmUtilityVersion `json:"utility-versions,omitempty"`
	Force                  bool                           `json:"force"`
}

// NewProvisionClusterRequestFromReader will create an UpdateClusterRequest from an io.Reader with JSON data.
func NewProvisionClusterRequestFromReader(reader io.Reader) (*ProvisionClusterRequest, error) {
	var provisionClusterRequest ProvisionClusterRequest
	err := json.NewDecoder(reader).Decode(&provisionClusterRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode provision cluster request")
	}

	return &provisionClusterRequest, nil
}
