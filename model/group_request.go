// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/pkg/errors"
)

const (
	forceInstallationRestartEnvVar = "CLOUD_PROVISIONER_ENFORCED_RESTART"

	// ShowInstallationCountQueryParameter the query parameter name for GET /groups in order to enable
	// or disable the installation count on the output.
	ShowInstallationCountQueryParameter = "show_installation_count"
)

// CreateGroupRequest specifies the parameters for a new group.
type CreateGroupRequest struct {
	Name            string
	Description     string
	Version         string
	Image           string
	MaxRolling      int64
	APISecurityLock bool
	MattermostEnv   EnvVarMap
	Annotations     []string
}

// Validate validates the values of a group create request.
func (request *CreateGroupRequest) Validate() error {
	if len(request.Name) == 0 {
		return errors.New("must specify name")
	}
	if request.MaxRolling < 0 {
		return errors.New("max rolling must be 0 or greater")
	}
	err := request.MattermostEnv.Validate()
	if err != nil {
		return errors.Wrapf(err, "bad environment variable map in create group request")
	}

	return nil
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

	return &createGroupRequest, nil
}

// PatchGroupRequest specifies the parameters for an updated group.
type PatchGroupRequest struct {
	ID            string
	MaxRolling    *int64
	Name          *string
	Description   *string
	Version       *string
	Image         *string
	MattermostEnv EnvVarMap

	ForceSequenceUpdate       bool
	ForceInstallationsRestart bool
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
	if p.MaxRolling != nil && *p.MaxRolling != group.MaxRolling {
		applied = true
		group.MaxRolling = *p.MaxRolling
	}
	if p.MattermostEnv != nil {
		if group.MattermostEnv.ClearOrPatch(&p.MattermostEnv) {
			applied = true
		}
	}

	// This special value allows us to bump the group sequence number even when
	// the patch contains no group modifications.
	if p.ForceSequenceUpdate {
		applied = true
	}

	// Force restart of pods even if nothing have changed. This is done by
	// setting non-meaningful environment variable.
	// We keep it separate from ForceSequenceUpdate in case we want to run force
	// update that does not require restarting pods.
	if p.ForceInstallationsRestart {
		group.MattermostEnv[forceInstallationRestartEnvVar] = EnvVar{Value: fmt.Sprintf("force-restart-at-sequence-%d", group.Sequence)}
		applied = true
	}

	return applied
}

// Validate validates the values of a group patch request
func (p *PatchGroupRequest) Validate() error {
	if p.Name != nil && len(*p.Name) == 0 {
		return errors.New("provided name update value was blank")
	}
	if p.MaxRolling != nil && *p.MaxRolling < 0 {
		return errors.New("max rolling must be 0 or greater")
	}
	// EnvVarMap validation is skipped as all configurations of this now imply
	// a specific patch action should be taken.

	return nil
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

// GetGroupsRequest describes the parameters to request a list of groups.
type GetGroupsRequest struct {
	Paging
	WithInstallationCount bool
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetGroupsRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	if request.WithInstallationCount {
		q.Add(ShowInstallationCountQueryParameter, "true")
	}
	request.Paging.AddToQuery(q)
	u.RawQuery = q.Encode()
}

// LeaveGroupRequest describes the parameters to leave a group.
type LeaveGroupRequest struct {
	RetainConfig bool
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *LeaveGroupRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	if !request.RetainConfig {
		q.Add("retain_config", "false")
	}
	u.RawQuery = q.Encode()
}
