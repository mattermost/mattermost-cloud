// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	mmv1alpha1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"
)

// CreateInstallationRequest specifies the parameters for a new installation.
type CreateInstallationRequest struct {
	OwnerID         string
	GroupID         string
	Version         string
	Image           string
	DNS             string
	License         string
	Size            string
	Affinity        string
	Database        string
	Filestore       string
	APISecurityLock bool
	MattermostEnv   EnvVarMap
	Annotations     []string
}

// https://man7.org/linux/man-pages/man7/hostname.7.html
var hostnamePattern *regexp.Regexp = regexp.MustCompile(`[a-zA-Z0-9][\.a-z-A-Z\-0-9]+`)

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
	if err := isValidDNS(request.DNS); err != nil {
		return err
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

	return checkSpaces(request)
}

func isValidDNS(dns string) error {
	if len(dns) > 253 {
		return errors.Errorf("fully qualified domain names must be less than 254 characters in length. Provided name %s was %d characters long", dns, len(dns))
	}
	subdomain := strings.SplitN(dns, ".", 2)[0]
	if len(subdomain) >= 64 || len(subdomain) < 3 {
		return errors.Errorf("DNS subdomain names must be between 3 and 64 characters, but name was %d long. DNS=%s", len(dns), dns)
	}
	// check that domain matches regex for valid names
	if found := hostnamePattern.FindString(dns); found != dns {
		return errors.Errorf("DNS name provided (%s) failed hostname pattern check", dns)
	}
	// check that domain does not resolve. Use a custom pure-Go resolver
	// to get the same behavior on VPN, in testing, and in production
	r := &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return (&net.Dialer{
				Timeout: time.Second * time.Duration(10000),
			}).DialContext(ctx, "udp", "1.1.1.1:53")
		},
	}
	_, err := r.LookupHost(context.Background(), dns)
	if err == nil {
		return errors.Errorf("dns name %s is already taken", dns)
	}
	switch e := err.(type) {
	case *net.DNSError:
		if !e.IsNotFound {
			return e
		}
		return nil
	default:
		// all of the errors that indicate success are DNSErrors. If
		// there's some other error, which shouldn't be possible, return
		// the unexpected error
		return errors.Wrapf(e, "unexpected error when looking up DNS name %s", dns)
	}
}

func checkSpaces(request *CreateInstallationRequest) error {
	if hasWhiteSpace(request.DNS) != -1 {
		return errors.Errorf("cannot have spaces in dns field. DNS=%s", request.DNS)
	}
	if hasWhiteSpace(request.Version) != -1 {
		return errors.Errorf("cannot have spaces in version field. Version=%s", request.Version)
	}
	if hasWhiteSpace(request.Image) != -1 {
		return errors.Errorf("cannot have spaces in image field. Image=%s", request.Image)
	}
	if hasWhiteSpace(request.License) != -1 {
		return errors.Errorf("cannot have spaces in license field. License=%s", request.License)
	}
	if hasWhiteSpace(request.GroupID) != -1 {
		return errors.Errorf("cannot have spaces in group field. Group=%s", request.GroupID)
	}

	return nil
}

func hasWhiteSpace(value string) int {
	return strings.IndexAny(value, " ")
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
	DNS                         string
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
	if request.DNS != "" {
		q.Add("dns_name", request.DNS)
	}
	u.RawQuery = q.Encode()
}

// PatchInstallationRequest specifies the parameters for an updated installation.
type PatchInstallationRequest struct {
	Version       *string
	Image         *string
	Size          *string
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
	if p.Size != nil {
		_, err := mmv1alpha1.GetClusterSize(*p.Size)
		if err != nil {
			return errors.Wrap(err, "invalid size")
		}
	}
	// EnvVarMap validation is skipped as all configurations of this now imply
	// a specific patch action should be taken.

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
	if p.Size != nil && *p.Size != installation.Size {
		applied = true
		installation.Size = *p.Size
	}
	if p.License != nil && *p.License != installation.License {
		applied = true
		installation.License = *p.License
	}
	if p.MattermostEnv != nil {
		if installation.MattermostEnv.ClearOrPatch(&p.MattermostEnv) {
			applied = true
		}
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
