package api

import (
	"encoding/json"
	"io"
	"net/url"
	"strconv"

	"github.com/mattermost/mattermost-cloud/internal/model"
	"github.com/pkg/errors"
)

// CreateInstallationRequest specifies the parameters for a new installation.
type CreateInstallationRequest struct {
	OwnerID  string
	Version  string
	DNS      string
	Affinity string
}

func newCreateInstallationRequestFromReader(reader io.Reader) (*CreateInstallationRequest, error) {
	var createInstallationRequest CreateInstallationRequest
	err := json.NewDecoder(reader).Decode(&createInstallationRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode create installation request")
	}

	if createInstallationRequest.Version == "" {
		createInstallationRequest.Version = "stable"
	}
	if createInstallationRequest.Affinity == "" {
		createInstallationRequest.Affinity = "isolated"
	}

	if createInstallationRequest.OwnerID == "" {
		return nil, errors.New("must specify owner")
	}
	if createInstallationRequest.DNS == "" {
		return nil, errors.New("must specify DNS")
	}
	if _, err := url.Parse(createInstallationRequest.DNS); err != nil {
		return nil, errors.Wrap(err, "invalid DNS")
	}
	if !model.IsSupportedAffinity(createInstallationRequest.Affinity) {
		return nil, errors.Errorf("unsupported affinity %s", createInstallationRequest.Affinity)
	}

	return &createInstallationRequest, nil
}

// GetInstallationsRequest describes the parameters to request a list of installations.
type GetInstallationsRequest struct {
	OwnerID        string
	Page           int
	PerPage        int
	IncludeDeleted bool
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetInstallationsRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("owner", request.OwnerID)
	q.Add("page", strconv.Itoa(request.Page))
	q.Add("per_page", strconv.Itoa(request.PerPage))
	if request.IncludeDeleted {
		q.Add("include_deleted", "true")
	}
	u.RawQuery = q.Encode()
}
