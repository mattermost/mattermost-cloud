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
	OwnerID   string
	Version   string
	DNS       string
	License   string
	Size      string
	Affinity  string
	Database  string
	Filestore string
}

// SetDefaults sets the default values for an installation create request.
func (request *CreateInstallationRequest) SetDefaults() {
	if request.Version == "" {
		request.Version = "stable"
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

// UpgradeInstallationRequest specifies the parameters for an upgraded installation.
type UpgradeInstallationRequest struct {
	Version string
	License string
}

// NewUpgradeInstallationRequestFromReader will create a UpgradeInstallationRequest from an io.Reader with JSON data.
func NewUpgradeInstallationRequestFromReader(reader io.Reader) (*UpgradeInstallationRequest, error) {
	var upgradeInstallationRequest UpgradeInstallationRequest
	err := json.NewDecoder(reader).Decode(&upgradeInstallationRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode upgrade installation request")
	}

	if upgradeInstallationRequest.Version == "" {
		return nil, errors.New("must specify version")
	}

	return &upgradeInstallationRequest, nil
}
