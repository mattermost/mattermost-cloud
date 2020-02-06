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
	Provider               string            `json:"provider,omitempty"`
	Version                string            `json:"version,omitempty"`
	KopsAMI                string            `json:"kops-ami,omitempty"`
	Size                   string            `json:"size,omitempty"`
	Zones                  []string          `json:"zones,omitempty"`
	AllowInstallations     bool              `json:"allow-installations,omitempty"`
	DesiredUtilityVersions map[string]string `json:"utility-versions,omitempty"`
}

// SetDefaults sets the default values for a cluster create request.
func (request *CreateClusterRequest) SetDefaults() {
	if request.Provider == "" {
		request.Provider = ProviderAWS
	}
	if request.Version == "" {
		request.Version = "latest"
	}
	if request.Size == "" {
		request.Size = SizeAlef500
	}
	if len(request.Zones) == 0 {
		request.Zones = []string{"us-east-1a"}
	}
	if request.DesiredUtilityVersions == nil {
		request.DesiredUtilityVersions = make(map[string]string)
	}
	if _, ok := request.DesiredUtilityVersions[PrometheusCanonicalName]; !ok {
		request.DesiredUtilityVersions[PrometheusCanonicalName] = PrometheusDefaultVersion
	}
	if _, ok := request.DesiredUtilityVersions[NginxCanonicalName]; !ok {
		request.DesiredUtilityVersions[NginxCanonicalName] = NginxDefaultVersion
	}
	if _, ok := request.DesiredUtilityVersions[FluentbitCanonicalName]; !ok {
		request.DesiredUtilityVersions[FluentbitCanonicalName] = FluentbitDefaultVersion
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
	if !IsSupportedClusterSize(request.Size) {
		return errors.Errorf("unsupported size %s", request.Size)
	}
	// TODO: check zones?

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

// ProvisionClusterRequest contains metadata related to changing the installed cluster state.
type ProvisionClusterRequest struct {
	DesiredUtilityVersions map[string]string `json:"utility-versions,omitempty"`
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
