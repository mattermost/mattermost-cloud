// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// requireAnnotatedInstallations if set, installations need to be annotated with at least one annotation.
var requireAnnotatedInstallations bool

// SetRequireAnnotatedInstallations is called with a value based on a CLI flag.
func SetRequireAnnotatedInstallations(val bool) {
	requireAnnotatedInstallations = val
}

// deployMySQLOperator if set, MySQL operator will be deployed
var deployMySQLOperator bool

// deployMinioOperator if set, Minio operator will be deployed
var deployMinioOperator bool

// SetDeployOperators is called with a value based on a CLI flag.
func SetDeployOperators(mysql, minio bool) {
	deployMySQLOperator = mysql
	deployMinioOperator = minio
}

// CreateInstallationRequest specifies the parameters for a new installation.
type CreateInstallationRequest struct {
	Name    string
	OwnerID string
	GroupID string
	Version string
	Image   string
	// Deprecated: Use DNSNames instead.
	DNS                       string
	DNSNames                  []string
	License                   string
	Size                      string
	Affinity                  string
	Database                  string
	Filestore                 string
	APISecurityLock           bool
	MattermostEnv             EnvVarMap
	PriorityEnv               EnvVarMap
	Annotations               []string
	GroupSelectionAnnotations []string
	// SingleTenantDatabaseConfig is ignored if Database is not single tenant mysql or postgres.
	SingleTenantDatabaseConfig SingleTenantDatabaseRequest
	// ExternalDatabaseConfig is ignored if Database is not single external.
	ExternalDatabaseConfig ExternalDatabaseRequest
}

// https://man7.org/linux/man-pages/man7/hostname.7.html
var hostnamePattern *regexp.Regexp = regexp.MustCompile(`[a-zA-Z0-9][\.a-z-A-Z\-0-9]+`)

// SetDefaults sets the default values for an installation create request.
func (request *CreateInstallationRequest) SetDefaults() {
	// If DNS is provided add it on the beginning of DNSNames slice.
	if request.DNS != "" {
		request.DNSNames = append([]string{request.DNS}, request.DNSNames...)
	}
	request.DNS = strings.ToLower(request.DNS)
	for i := range request.DNSNames {
		request.DNSNames[i] = strings.ToLower(request.DNSNames[i])
	}

	// For backwards compatibility set Name based on DNS
	if request.Name == "" && request.DNS != "" {
		request.Name = strings.Split(request.DNS, ".")[0]
	}
	request.Name = strings.ToLower(request.Name)
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
	if IsSingleTenantRDS(request.Database) {
		request.SingleTenantDatabaseConfig.SetDefaults()
	}
}

// Validate validates the values of an installation create request.
func (request *CreateInstallationRequest) Validate() error {
	if request.Name == "" {
		return errors.New("name needs to be specified")
	}
	if request.OwnerID == "" {
		return errors.New("must specify owner")
	}

	err := request.validateDNSNames()
	if err != nil {
		return err
	}

	_, err = GetInstallationSize(request.Size)
	if err != nil {
		return errors.Wrap(err, "invalid Installation size")
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
	err = request.PriorityEnv.Validate()
	if err != nil {
		return errors.Wrap(err, "invalid priority env var settings")
	}

	if requireAnnotatedInstallations {
		if len(request.Annotations) == 0 {
			return errors.Errorf("at least one annotation is required")
		}
	}

	for _, ann := range request.GroupSelectionAnnotations {
		errInner := validateAnnotationName(ann)
		if errInner != nil {
			return errors.Wrap(errInner, "invalid group selection annotation")
		}
	}

	if IsSingleTenantRDS(request.Database) {
		err = request.SingleTenantDatabaseConfig.Validate()
		if err != nil {
			return errors.Wrap(err, "single tenant database config is invalid")
		}
	}

	if request.Database == InstallationDatabaseExternal {
		err = request.ExternalDatabaseConfig.Validate()
		if err != nil {
			return errors.Wrap(err, "external database config is invalid")
		}
	}

	if !deployMinioOperator && request.Filestore == InstallationFilestoreMinioOperator {
		return errors.Errorf("minio filestore cannot be used when minio operator is not deployed")
	}
	if !deployMySQLOperator && request.Database == InstallationDatabaseMysqlOperator {
		return errors.Errorf("mysql operator database cannot be used when mysql operator is not deployed")
	}
	return checkSpaces(request)
}

func (request *CreateInstallationRequest) validateDNSNames() error {
	if len(request.DNSNames) == 0 {
		return errors.New("at least one DNS name is required")
	}

	for _, dns := range request.DNSNames {
		if err := isValidDNS(dns); err != nil {
			return err
		}

		err := ensureDNSMatchesName(dns, request.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func ensureDNSMatchesName(dns string, name string) error {
	dnsPrefix := strings.Split(dns, ".")[0]
	if dnsPrefix != name {
		return errors.Errorf("domain name must start with Installation Name")
	}
	return nil
}

func isValidDNS(dns string) error {
	if len(dns) > 253 {
		return errors.Errorf("fully qualified domain names must be less than 254 characters in length. Provided name %s was %d characters long", dns, len(dns))
	}
	subdomain := strings.SplitN(dns, ".", 2)[0]
	if len(subdomain) >= 64 || len(subdomain) < 2 {
		return errors.Errorf("DNS subdomain names must be between 3 and 64 characters, but name was %d long. DNS=%s", len(dns), dns)
	}
	// check that domain matches regex for valid names
	if found := hostnamePattern.FindString(dns); found != dns {
		return errors.Errorf("DNS name provided (%s) failed hostname pattern check", dns)
	}
	return nil
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
	Paging
	OwnerID                     string
	GroupID                     string
	State                       string
	DNS                         string
	Name                        string
	IncludeGroupConfig          bool
	IncludeGroupConfigOverrides bool
}

// ApplyToURL modifies the given url to include query string parameters for the request.
func (request *GetInstallationsRequest) ApplyToURL(u *url.URL) {
	q := u.Query()
	q.Add("owner", request.OwnerID)
	q.Add("group", request.GroupID)
	q.Add("state", request.State)
	q.Add("dns_name", request.DNS)
	q.Add("name", request.Name)
	if !request.IncludeGroupConfig {
		q.Add("include_group_config", "false")
	}
	if !request.IncludeGroupConfigOverrides {
		q.Add("include_group_config_overrides", "false")
	}
	request.Paging.AddToQuery(q)

	u.RawQuery = q.Encode()
}

// PatchInstallationRequest specifies the parameters for an updated installation.
type PatchInstallationRequest struct {
	OwnerID       *string
	Image         *string
	Version       *string
	Size          *string
	License       *string
	PriorityEnv   EnvVarMap
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
		_, err := GetInstallationSize(*p.Size)
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

	if p.OwnerID != nil && *p.OwnerID != installation.OwnerID {
		applied = true
		installation.OwnerID = *p.OwnerID
	}
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
	if p.PriorityEnv != nil {
		if installation.PriorityEnv.ClearOrPatch(&p.PriorityEnv) {
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

// PatchInstallationDeletionRequest specifies the parameters for an updating
// installation deletion parameters.
type PatchInstallationDeletionRequest struct {
	DeletionPendingExpiry *int64
}

// Validate validates the values of a installation deletion patch request.
func (p *PatchInstallationDeletionRequest) Validate() error {
	if p.DeletionPendingExpiry != nil {
		// DeletionPendingExpiry is the new time when an installation pending
		// deletion can be deleted. This can be any time from "now" into the
		// future. The cuttoff for "now" will be the current time with a 5 second
		// buffer. Any time value lower than that will be considered an error.
		cutoffTimeMillis := GetMillisAtTime(time.Now().Add(-5 * time.Second))
		if cutoffTimeMillis > *p.DeletionPendingExpiry {
			return errors.Errorf("DeletionPendingExpiry must be %d or higher", cutoffTimeMillis)
		}
	}

	return nil
}

// Apply applies the deletion patch to the given installation.
func (p *PatchInstallationDeletionRequest) Apply(installation *Installation) bool {
	var applied bool

	if p.DeletionPendingExpiry != nil && *p.DeletionPendingExpiry != installation.DeletionPendingExpiry {
		applied = true
		installation.DeletionPendingExpiry = *p.DeletionPendingExpiry
	}

	return applied
}

// NewPatchInstallationDeletionRequestFromReader will create a PatchInstallationDeletionRequest from an io.Reader with JSON data.
func NewPatchInstallationDeletionRequestFromReader(reader io.Reader) (*PatchInstallationDeletionRequest, error) {
	var patchInstallationDeletionRequest PatchInstallationDeletionRequest
	err := json.NewDecoder(reader).Decode(&patchInstallationDeletionRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode patch installation deletion request")
	}

	err = patchInstallationDeletionRequest.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "invalid patch installation deletion request")
	}

	return &patchInstallationDeletionRequest, nil
}

// AssignInstallationGroupRequest specifies request body for installation group assignment.
type AssignInstallationGroupRequest struct {
	GroupSelectionAnnotations []string
}

// NewAssignInstallationGroupRequestFromReader will create a AssignInstallationGroupRequest from an io.Reader with JSON data.
func NewAssignInstallationGroupRequestFromReader(reader io.Reader) (*AssignInstallationGroupRequest, error) {
	var assignGroupRequest AssignInstallationGroupRequest
	err := json.NewDecoder(reader).Decode(&assignGroupRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode assign group request")
	}

	return &assignGroupRequest, nil
}
