package model

import (
	"encoding/json"
	"io"
	"net/url"
	"strconv"

	"github.com/pkg/errors"
)

// GetClusterInstallationsRequest describes the parameters to request a list of cluster installations.
type GetClusterInstallationsRequest struct {
	ClusterID      string
	InstallationID string
	Page           int
	PerPage        int
	IncludeDeleted bool
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetClusterInstallationsRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("cluster", request.ClusterID)
	q.Add("installation", request.InstallationID)
	q.Add("page", strconv.Itoa(request.Page))
	q.Add("per_page", strconv.Itoa(request.PerPage))
	if request.IncludeDeleted {
		q.Add("include_deleted", "true")
	}
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
