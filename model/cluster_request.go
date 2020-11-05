// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
	"net/url"
	"strconv"

	"github.com/pkg/errors"
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
	if request.DesiredUtilityVersions == nil {
		request.DesiredUtilityVersions = make(map[string]*HelmUtilityVersion)
	}
	if _, ok := request.DesiredUtilityVersions[PrometheusOperatorCanonicalName]; !ok {
		request.DesiredUtilityVersions[PrometheusOperatorCanonicalName] = PrometheusOperatorDefaultVersion
	}
	if _, ok := request.DesiredUtilityVersions[ThanosCanonicalName]; !ok {
		request.DesiredUtilityVersions[ThanosCanonicalName] = ThanosDefaultVersion
	}
	if _, ok := request.DesiredUtilityVersions[NginxCanonicalName]; !ok {
		request.DesiredUtilityVersions[NginxCanonicalName] = NginxDefaultVersion
	}
	if _, ok := request.DesiredUtilityVersions[NginxInternalCanonicalName]; !ok {
		request.DesiredUtilityVersions[NginxInternalCanonicalName] = NginxIntenalDefaultVersion
	}
	if _, ok := request.DesiredUtilityVersions[FluentbitCanonicalName]; !ok {
		request.DesiredUtilityVersions[FluentbitCanonicalName] = FluentbitDefaultVersion
	}
	if _, ok := request.DesiredUtilityVersions[TeleportCanonicalName]; !ok {
		request.DesiredUtilityVersions[TeleportCanonicalName] = TeleportDefaultVersion
	}
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
	// TODO: check zones and instance types?

	return nil
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
	Page           int
	PerPage        int
	IncludeDeleted bool
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetClustersRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("page", strconv.Itoa(request.Page))
	q.Add("per_page", strconv.Itoa(request.PerPage))
	if request.IncludeDeleted {
		q.Add("include_deleted", "true")
	}
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
	Version *string `json:"version,omitempty"`
	KopsAMI *string `json:"kops-ami,omitempty"`
}

// Validate validates the values of a cluster upgrade request.
func (p *PatchUpgradeClusterRequest) Validate() error {
	if p.Version != nil && !ValidClusterVersion(*p.Version) {
		return errors.Errorf("unsupported cluster version %s", *p.Version)
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

	if applied {
		metadata.ChangeRequest = changes
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
	NodeInstanceType *string `json:"node-instance-type,omitempty"`
	NodeMinCount     *int64  `json:"node-min-count,omitempty"`
	NodeMaxCount     *int64  `json:"node-max-count,omitempty"`
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

	if applied {
		metadata.ChangeRequest = changes
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
