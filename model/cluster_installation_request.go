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

// GetClusterInstallationsRequest describes the parameters to request a list of cluster installations.
type GetClusterInstallationsRequest struct {
	Paging
	ClusterID      string
	InstallationID string
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetClusterInstallationsRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("cluster", request.ClusterID)
	q.Add("installation", request.InstallationID)
	request.Paging.AddToQuery(q)

	u.RawQuery = q.Encode()
}

// ClusterInstallationConfigRequest describes the payload for updating an cluster installation's configuration.
type ClusterInstallationConfigRequest map[string]interface{}

// NewClusterInstallationConfigRequestFromReader will create a ClusterInstallationConfigRequest from an io.Reader with JSON data.
func NewClusterInstallationConfigRequestFromReader(reader io.Reader) (ClusterInstallationConfigRequest, error) {
	var clusterInstallationConfigRequest ClusterInstallationConfigRequest
	err := json.NewDecoder(reader).Decode(&clusterInstallationConfigRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode cluster installation config request")
	}

	return clusterInstallationConfigRequest, nil
}

// ClusterInstallationMattermostCLISubcommand describes the payload necessary to run Mattermost CLI on a cluster installation.
type ClusterInstallationMattermostCLISubcommand []string

// NewClusterInstallationMattermostCLISubcommandFromReader will create a ClusterInstallationMattermostCLISubcommand from an io.Reader.
func NewClusterInstallationMattermostCLISubcommandFromReader(reader io.Reader) (ClusterInstallationMattermostCLISubcommand, error) {
	var clusterInstallationMattermostCLISubcommand ClusterInstallationMattermostCLISubcommand
	err := json.NewDecoder(reader).Decode(&clusterInstallationMattermostCLISubcommand)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode cluster installation mattermost CLI request")
	}

	return clusterInstallationMattermostCLISubcommand, nil
}

// ClusterInstallationExecSubcommand describes the payload necessary to run container exec commands on a cluster installation.
type ClusterInstallationExecSubcommand []string

// NewClusterInstallationExecSubcommandFromReader will create a ClusterInstallationExecSubcommand from an io.Reader.
func NewClusterInstallationExecSubcommandFromReader(reader io.Reader) (ClusterInstallationExecSubcommand, error) {
	var clusterInstallationExecSubcommand ClusterInstallationExecSubcommand
	err := json.NewDecoder(reader).Decode(&clusterInstallationExecSubcommand)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode cluster installation exec request")
	}

	return clusterInstallationExecSubcommand, nil
}

// NewMigrateClusterInstallationRequestFromReader will create a MigrateClusterInstallationRequest from an io.Reader.
func NewMigrateClusterInstallationRequestFromReader(reader io.Reader) (MigrateClusterInstallationRequest, error) {
	var migrateClusterInstallationRequest MigrateClusterInstallationRequest
	err := json.NewDecoder(reader).Decode(&migrateClusterInstallationRequest)
	if err != nil && err != io.EOF {
		return MigrateClusterInstallationRequest{}, errors.Wrap(err, "failed to decode cluster installation migration request")
	}

	err = migrateClusterInstallationRequest.Validate()
	if err != nil {
		return MigrateClusterInstallationRequest{}, errors.Wrap(err, "migrate cluster installation request failed validation")
	}
	return migrateClusterInstallationRequest, nil
}

// Validate validate the migration request for cluster installations
func (request *MigrateClusterInstallationRequest) Validate() error {
	if len(request.SourceClusterID) == 0 {
		return errors.New("missing mandatory source cluster in a migration request")
	}

	if len(request.TargetClusterID) == 0 {
		return errors.New("missing mandatory target cluster in a migration request")
	}

	return nil
}
