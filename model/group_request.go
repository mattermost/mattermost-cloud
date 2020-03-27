package model

import (
	"encoding/json"
	"io"
	"net/url"
	"strconv"

	"github.com/pkg/errors"
)

// CreateGroupRequest specifies the parameters for a new group.
type CreateGroupRequest struct {
	Name          string
	Description   string
	Version       string
	Image         string
	MattermostEnv EnvVarMap
}

// NewCreateGroupRequestFromReader will create a CreateGroupRequest from an io.Reader with JSON data.
func NewCreateGroupRequestFromReader(reader io.Reader) (*CreateGroupRequest, error) {
	var createGroupRequest CreateGroupRequest
	err := json.NewDecoder(reader).Decode(&createGroupRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode create group request")
	}

	err = createGroupRequest.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "invalid group create request")
	}
	if createGroupRequest.Image == "" {
		return nil, errors.New("must specify image")
	}

	return &createGroupRequest, nil
}

// Validate validates the values of a group create request
func (request *CreateGroupRequest) Validate() error {
	if request.Name == "" {
		return errors.New("must specify name")
	}
	if request.Version == "" {
		return errors.New("must specify version")
	}
	err := request.MattermostEnv.Validate()
	if err != nil {
		return errors.Wrapf(err, "bad environment variable map in create group request")
	}

	return nil
}

// PatchGroupRequest specifies the parameters for an updated group.
type PatchGroupRequest struct {
	ID            string
	Name          *string
	Description   *string
	Version       *string
	Image         *string
	MattermostEnv EnvVarMap
}

// NewPatchGroupRequestFromReader will create a PatchGroupRequest from an io.Reader with JSON data.
func NewPatchGroupRequestFromReader(reader io.Reader) (*PatchGroupRequest, error) {
	var patchGroupRequest PatchGroupRequest
	err := json.NewDecoder(reader).Decode(&patchGroupRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode patch group request")
	}

	err = patchGroupRequest.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "invalid patch group request")
	}

	return &patchGroupRequest, nil
}

// Apply applies the patch to the given group.
func (p *PatchGroupRequest) Apply(group *Group) bool {
	var applied bool

	if p.Name != nil && *p.Name != group.Name {
		applied = true
		group.Name = *p.Name
	}
	if p.Description != nil && *p.Description != group.Description {
		applied = true
		group.Description = *p.Description
	}
	if p.Version != nil && *p.Version != group.Version {
		applied = true
		group.Version = *p.Version
	}
	if p.Image != nil && *p.Image != group.Image {
		applied = true
		group.Image = *p.Image
	}

	if p.MattermostEnv != nil {
		applied = true
		group.MattermostEnv = p.MattermostEnv
	}

	return applied
}

// Validate validates the values of a group patch request
func (p *PatchGroupRequest) Validate() error {
	err := p.MattermostEnv.Validate()
	if err != nil {
		return errors.Wrapf(err, "bad environment variable map in patch group request")
	}

	return nil
}

// GetGroupsRequest describes the parameters to request a list of groups.
type GetGroupsRequest struct {
	Page           int
	PerPage        int
	IncludeDeleted bool
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetGroupsRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("page", strconv.Itoa(request.Page))
	q.Add("per_page", strconv.Itoa(request.PerPage))
	if request.IncludeDeleted {
		q.Add("include_deleted", "true")
	}
	u.RawQuery = q.Encode()
}
