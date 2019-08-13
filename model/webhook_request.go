package model

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"

	"github.com/pkg/errors"
)

// CreateWebhookRequest specifies the parameters for a new webhook.
type CreateWebhookRequest struct {
	OwnerID string
	URL     string
}

// NewCreateWebhookRequestFromReader will create a CreateWebhookRequest from an io.Reader with JSON data.
func NewCreateWebhookRequestFromReader(reader io.Reader) (*CreateWebhookRequest, error) {
	var createWebhookRequest CreateWebhookRequest
	err := json.NewDecoder(reader).Decode(&createWebhookRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode create webhook request")
	}

	if createWebhookRequest.OwnerID == "" {
		return nil, errors.New("must specify owner")
	}
	if createWebhookRequest.URL == "" {
		return nil, errors.New("must specify callback URL")
	}
	uri, err := url.ParseRequestURI(createWebhookRequest.URL)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse callback URL")
	}
	switch uri.Scheme {
	case "http", "https":
	default:
		return nil, fmt.Errorf("'%s' is not a valid scheme: should be 'http' or 'https'", uri.Scheme)
	}
	if uri.Host == "" {
		return nil, errors.New("must specify host")
	}

	return &createWebhookRequest, nil
}

// GetWebhooksRequest describes the parameters to request a list of webhooks.
type GetWebhooksRequest struct {
	OwnerID        string
	Page           int
	PerPage        int
	IncludeDeleted bool
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetWebhooksRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("owner", request.OwnerID)
	q.Add("page", strconv.Itoa(request.Page))
	q.Add("per_page", strconv.Itoa(request.PerPage))
	if request.IncludeDeleted {
		q.Add("include_deleted", "true")
	}
	u.RawQuery = q.Encode()
}
