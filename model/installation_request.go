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

// PatchInstallationRequest specifies the parameters for an updated installation.
type PatchInstallationRequest struct {
	Version       *string
	Image         *string
	License       *string
	MattermostEnv EnvVarMap
}

// Validate validates the values of a installation patch request.
func (p *PatchInstallationRequest) Validate() error {
	if p.Version != nil && len(*p.Version) == 0 {
		return errors.New("provided version update value was blank")
	}
	if p.Image != nil && len(*p.Image) == 0 {
		return errors.New("provided image update value was blank")
	}
	err := p.MattermostEnv.Validate()
	if err != nil {
		return errors.Wrap(err, "invalid env var settings")
	}

	return nil
}

// Apply applies the patch to the given installation.
func (p *PatchInstallationRequest) Apply(installation *Installation) bool {
	var applied bool

	if p.Version != nil && *p.Version != installation.Version {
		applied = true
		installation.Version = *p.Version
	}
	if p.Image != nil && *p.Image != installation.Image {
		applied = true
		installation.Image = *p.Image
	}
	if p.License != nil && *p.License != installation.License {
		applied = true
		installation.License = *p.License
	}
	if p.MattermostEnv != nil {
		applied = true
		installation.MattermostEnv = p.MattermostEnv
	}

	return applied
}

// NewPatchInstallationRequestFromReader will create a PatchInstallationRequest from an io.Reader with JSON data.
func NewPatchInstallationRequestFromReader(reader io.Reader) (*PatchInstallationRequest, error) {
	var patchInstallationRequest PatchInstallationRequest
	err := json.NewDecoder(reader).Decode(&patchInstallationRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode patch installation request")
	}

	err = patchInstallationRequest.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "invalid patch installation request")
	}

	return &patchInstallationRequest, nil
}
