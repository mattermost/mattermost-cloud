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
	// NetworkingVpcCni is Amazon VPC CNI networking plugin.
	NetworkingVpcCni = "amazon-vpc-cni"
)

// CreateClusterRequest specifies the parameters for a new cluster.
type CreateClusterRequest struct {
	Provider                   string                         `json:"provider,omitempty"`
	Zones                      []string                       `json:"zones,omitempty"`
	Version                    string                         `json:"version,omitempty"`
	AMI                        string                         `json:"ami,omitempty"`
	MasterInstanceType         string                         `json:"master-instance-type,omitempty"`
	MasterCount                int64                          `json:"master-count,omitempty"`
	NodeInstanceType           string                         `json:"node-instance-type,omitempty"`
	NodeMinCount               int64                          `json:"node-min-count,omitempty"`
	NodeMaxCount               int64                          `json:"node-max-count,omitempty"`
	AllowInstallations         bool                           `json:"allow-installations,omitempty"`
	APISecurityLock            bool                           `json:"api-security-lock,omitempty"`
	DesiredUtilityVersions     map[string]*HelmUtilityVersion `json:"utility-versions,omitempty"`
	Annotations                []string                       `json:"annotations,omitempty"`
	Networking                 string                         `json:"networking,omitempty"`
	VPC                        string                         `json:"vpc,omitempty"`
	MaxPodsPerNode             int64                          `json:"max-pods-per-node,omitempty"`
	ClusterRoleARN             string                         `json:"cluster-role-arn,omitempty"`
	NodeRoleARN                string                         `json:"node-role-arn,omitempty"`
	Provisioner                string                         `json:"provisioner,omitempty"`
	AdditionalNodeGroups       map[string]NodeGroupMetadata   `json:"additional-node-groups,omitempty"`
	NodeGroupWithPublicSubnet  []string                       `json:"nodegroup-with-public-subnet,omitempty"`
	NodeGroupWithSecurityGroup []string                       `json:"nodegroup-with-sg,omitempty"`
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
	if len(request.Provisioner) == 0 {
		request.Provisioner = ProvisionerKops
	}
	if len(request.Version) == 0 {
		if request.Provisioner == ProvisionerEKS {
			request.Version = "1.23"
		} else {
			request.Version = "latest"
		}
	}

	if len(request.Zones) == 0 {
		if request.Provisioner == ProvisionerEKS {
			request.Zones = []string{"us-east-1a", "us-east-1b"}
		} else {
			request.Zones = []string{"us-east-1a"}
		}
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

	if request.Provisioner == ProvisionerEKS {
		for ng, meta := range request.AdditionalNodeGroups {
			if len(meta.InstanceType) == 0 {
				meta.InstanceType = "m5.large"
			}
			if meta.MinCount == 0 {
				meta.MinCount = 2
			}
			if meta.MaxCount == 0 {
				meta.MaxCount = meta.MinCount
			}

			request.AdditionalNodeGroups[ng] = meta
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
	if request.Provisioner != ProvisionerKops && request.Provisioner != ProvisionerEKS {
		return errors.Errorf("unsupported provisioner %s", request.Provisioner)
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

	if request.Provisioner == ProvisionerEKS {
		if request.ClusterRoleARN == "" {
			return errors.New("cluster role ARN for EKS cluster cannot be empty")
		}
		if request.NodeRoleARN == "" {
			return errors.New("node role ARN for EKS cluster cannot be empty")
		}
		if request.AMI == "" {
			return errors.New("AMI for EKS cluster cannot be empty")
		}

		if len(request.Zones) < 2 {
			return errors.New("EKS cluster needs at least two zones")
		}

		if request.AdditionalNodeGroups != nil {
			if _, f := request.AdditionalNodeGroups[NodeGroupWorker]; f {
				return errors.New("additional node group name cannot be named worker")
			}

			for name, ng := range request.AdditionalNodeGroups {
				if ng.MinCount < 1 {
					return errors.Errorf("node min count (%d) must be 1 or greater for node group %s", ng.MinCount, name)
				}
				if ng.MaxCount != ng.MinCount {
					return errors.Errorf("node min (%d) and max (%d) counts must match for node group %s", ng.MinCount, ng.MaxCount, name)
				}
			}
		}

		for _, ng := range request.NodeGroupWithPublicSubnet {
			if ng == NodeGroupWorker {
				continue
			}
			if _, f := request.AdditionalNodeGroups[ng]; !f {
				return errors.Errorf("invalid nodegroup %s to use public subnets", ng)
			}
		}

		for _, ng := range request.NodeGroupWithSecurityGroup {
			if ng == NodeGroupWorker {
				continue
			}
			if _, f := request.AdditionalNodeGroups[ng]; !f {
				return errors.Errorf("invalid nodegroup %s to use security group", ng)
			}
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
	AMI            *string        `json:"ami,omitempty"`
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
	NodeGroups       []string       `json:"nodeGroups,omitempty"`
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

type CreateNodegroupsRequest struct {
	Nodegroups                 map[string]NodeGroupMetadata `json:"nodegroups"`
	NodeGroupWithPublicSubnet  []string                     `json:"nodegroup-with-public-subnet,omitempty"`
	NodeGroupWithSecurityGroup []string                     `json:"nodegroup-with-sg,omitempty"`
}

// SetDefaults sets default values for nodegroups.
func (request *CreateNodegroupsRequest) SetDefaults() {
	for ng, meta := range request.Nodegroups {
		if len(meta.InstanceType) == 0 {
			meta.InstanceType = "m5.large"
		}
		if meta.MinCount == 0 {
			meta.MinCount = 2
		}
		if meta.MaxCount == 0 {
			meta.MaxCount = meta.MinCount
		}

		request.Nodegroups[ng] = meta
	}
}

// Validate validates the values of a nodegroup creation request.
func (request *CreateNodegroupsRequest) Validate() error {
	for ng, meta := range request.Nodegroups {
		if meta.MinCount < 1 {
			return errors.Errorf("nodegroup %s min count has to be 1 or greater", ng)
		}
		if meta.MaxCount < meta.MinCount {
			return errors.Errorf("nodegroup %s max count (%d) can't be less than min count (%d)", ng, meta.MaxCount, meta.MinCount)
		}
	}

	for _, ng := range request.NodeGroupWithPublicSubnet {
		if ng == NodeGroupWorker {
			continue
		}
		if _, f := request.Nodegroups[ng]; !f {
			return errors.Errorf("invalid nodegroup %s to use public subnets", ng)
		}
	}

	for _, ng := range request.NodeGroupWithSecurityGroup {
		if ng == NodeGroupWorker {
			continue
		}
		if _, f := request.Nodegroups[ng]; !f {
			return errors.Errorf("invalid nodegroup %s to use security group", ng)
		}
	}

	return nil
}

// NewCreateNodegroupsRequestFromReader will create an CreateNodegroupsRequest from an io.Reader with JSON data.
func NewCreateNodegroupsRequestFromReader(reader io.Reader) (*CreateNodegroupsRequest, error) {
	var createNodegroupsRequest CreateNodegroupsRequest
	err := json.NewDecoder(reader).Decode(&createNodegroupsRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode create nodegroups request")
	}

	createNodegroupsRequest.SetDefaults()
	err = createNodegroupsRequest.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "create nodegroups request failed validation")
	}

	return &createNodegroupsRequest, nil
}
