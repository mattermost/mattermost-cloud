package api

import (
	"encoding/json"
	"io"
	"net/url"
	"strconv"

	"github.com/mattermost/mattermost-cloud/internal/model"
	"github.com/pkg/errors"
)

// CreateClusterRequest specifies the parameters for a new cluster.
type CreateClusterRequest struct {
	Provider string
	Size     string
	Zones    []string
}

func newCreateClusterRequestFromReader(reader io.Reader) (*CreateClusterRequest, error) {
	var createClusterRequest CreateClusterRequest
	err := json.NewDecoder(reader).Decode(&createClusterRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode create cluster request")
	}

	if createClusterRequest.Provider == "" {
		createClusterRequest.Provider = model.ProviderAWS
	}
	if createClusterRequest.Size == "" {
		createClusterRequest.Size = model.SizeAlef500
	}
	if len(createClusterRequest.Zones) == 0 {
		createClusterRequest.Zones = []string{"us-east-1a"}
	}

	if createClusterRequest.Provider != model.ProviderAWS {
		return nil, errors.Errorf("unsupported provider %s", createClusterRequest.Provider)
	}
	if !model.IsSupportedSize(createClusterRequest.Size) {
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
