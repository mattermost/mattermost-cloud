package model

import (
	"encoding/json"
	"io"
	"net/url"
	"strconv"

	"github.com/pkg/errors"

	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
)

// CreateInstallationRequest specifies the parameters for a new installation.
type CreateInstallationRequest struct {
	OwnerID       string
	GroupID       string
	Version       string
	Image         string
	DNS           string
	License       string
	Size          string
	Affinity      string
	Database      string
	Filestore     string
	MattermostEnv EnvVarMap
}

// SetDefaults sets the default values for an installation create request.
func (request *CreateInstallationRequest) SetDefaults() {
	if request.Version == "" {
		request.Version = "stable"
	}
	if request.Image == "" {
		request.Image = "mattermost/mattermost-enterprise-edition"
	}
	if request.Size == "" {
		request.Size = InstallationDefaultSize
	}
	if request.Affinity == "" {
		request.Affinity = InstallationAffinityIsolated
	}
	if request.Database == "" {
		request.Database = InstallationDatabaseMysqlOperator
	}
	if request.Filestore == "" {
		request.Filestore = InstallationFilestoreMinioOperator
	}
}

// Validate validates the values of an installation create request.
func (request *CreateInstallationRequest) Validate() error {
	if request.OwnerID == "" {
		return errors.New("must specify owner")
	}
	if request.DNS == "" {
		return errors.New("must specify DNS")
	}
	if len(request.DNS) >= 64 {
		return errors.Errorf("DNS names must be less than 64 characters, but name was %d long. DNS=%s", len(request.DNS), request.DNS)
	}
	_, err := mmv1alpha1.GetClusterSize(request.Size)
	if err != nil {
		return errors.Wrap(err, "invalid size")
	}
	_, err = url.Parse(request.DNS)
	if err != nil {
		return errors.Wrapf(err, "invalid DNS %s", request.DNS)
	}
	if !IsSupportedAffinity(request.Affinity) {
		return errors.Errorf("unsupported affinity %s", request.Affinity)
	}
	if !IsSupportedDatabase(request.Database) {
		return errors.Errorf("unsupported database %s", request.Database)
	}
	if !IsSupportedFilestore(request.Filestore) {
		return errors.Errorf("unsupported filestore %s", request.Filestore)
	}
	err = request.MattermostEnv.Validate()
	if err != nil {
		return errors.Wrap(err, "invalid env var settings")
	}

	return nil
}

// NewCreateInstallationRequestFromReader will create a CreateInstallationRequest from an io.Reader with JSON data.
func NewCreateInstallationRequestFromReader(reader io.Reader) (*CreateInstallationRequest, error) {
	var createInstallationRequest CreateInstallationRequest
	err := json.NewDecoder(reader).Decode(&createInstallationRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode create installation request")
	}

	createInstallationRequest.SetDefaults()
	err = createInstallationRequest.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "create installation request failed validation")
	}

	return &createInstallationRequest, nil
}

// GetInstallationRequest describes the parameters to request an installation.
type GetInstallationRequest struct {
	IncludeGroupConfig          bool
	IncludeGroupConfigOverrides bool
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetInstallationRequest) ApplyToURL(u *url.URL) {
	if request == nil {
		return
	}
	q := u.Query()
	if !request.IncludeGroupConfig {
		q.Add("include_group_config", "false")
	}
	if !request.IncludeGroupConfigOverrides {
		q.Add("include_group_config_overrides", "false")
	}
	u.RawQuery = q.Encode()
}

// GetInstallationsRequest describes the parameters to request a list of installations.
type GetInstallationsRequest struct {
	OwnerID                     string
	GroupID                     string
	IncludeGroupConfig          bool
	IncludeGroupConfigOverrides bool
	Page                        int
	PerPage                     int
	IncludeDeleted              bool
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetInstallationsRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("owner", request.OwnerID)
	q.Add("group", request.GroupID)
	if !request.IncludeGroupConfig {
		q.Add("include_group_config", "false")
	}
	if !request.IncludeGroupConfigOverrides {
		q.Add("include_group_config_overrides", "false")
	}
	q.Add("page", strconv.Itoa(request.Page))
	q.Add("per_page", strconv.Itoa(request.PerPage))
	if request.IncludeDeleted {
		q.Add("include_deleted", "true")
	}
	u.RawQuery = q.Encode()
}

// UpdateInstallationRequest specifies the parameters for an updated installation.
type UpdateInstallationRequest struct {
	Version       string
	Image         string
	License       string
	MattermostEnv EnvVarMap
}

// SetDefaults sets the default values for an installation update request.
func (request *UpdateInstallationRequest) SetDefaults() {
	if request.Image == "" {
		request.Image = "mattermost/mattermost-enterprise-edition"
	}
}

// Validate validates the values of an installation update request.
func (request *UpdateInstallationRequest) Validate() error {
	// TODO: remove version check after verifying that nothing will break.
	if request.Version == "" {
		return errors.New("must specify version")
	}
	err := request.MattermostEnv.Validate()
	if err != nil {
		return errors.Wrap(err, "invalid env var settings")
	}

	return nil
}

// NewUpdateInstallationRequestFromReader will create a UpdateInstallationRequest from an io.Reader with JSON data.
func NewUpdateInstallationRequestFromReader(reader io.Reader) (*UpdateInstallationRequest, error) {
	var updateInstallationRequest UpdateInstallationRequest
	err := json.NewDecoder(reader).Decode(&updateInstallationRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode upgrade installation request")
	}

	updateInstallationRequest.SetDefaults()
	err = updateInstallationRequest.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "update installation request failed validation")
	}

	return &updateInstallationRequest, nil
}
