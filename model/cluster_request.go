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
	Provider           string   `json:"provider,omitempty"`
	Version            string   `json:"version,omitempty"`
	KopsAMI            string   `json:"kops-ami,omitempty"`
	Size               string   `json:"size,omitempty"`
	Zones              []string `json:"zones,omitempty"`
	AllowInstallations bool     `json:"allow-installations,omitempty"`
}

// NewCreateClusterRequestFromReader will create a CreateClusterRequest from an io.Reader with JSON data.
func NewCreateClusterRequestFromReader(reader io.Reader) (*CreateClusterRequest, error) {
	var createClusterRequest CreateClusterRequest
	err := json.NewDecoder(reader).Decode(&createClusterRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode create cluster request")
	}

	if createClusterRequest.Provider == "" {
		createClusterRequest.Provider = ProviderAWS
	}
	if createClusterRequest.Version == "" {
		createClusterRequest.Version = "latest"
	}
	if createClusterRequest.Size == "" {
		createClusterRequest.Size = SizeAlef500
	}
	if len(createClusterRequest.Zones) == 0 {
		createClusterRequest.Zones = []string{"us-east-1a"}
	}

	if createClusterRequest.Provider != ProviderAWS {
		return nil, errors.Errorf("unsupported provider %s", createClusterRequest.Provider)
	}
	if !ValidClusterVersion(createClusterRequest.Version) {
		return nil, errors.Errorf("unsupported cluster version %s", createClusterRequest.Version)
	}
	if !IsSupportedClusterSize(createClusterRequest.Size) {
		return nil, errors.Errorf("unsupported size %s", createClusterRequest.Size)
	}
	// TODO: check zones?

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
