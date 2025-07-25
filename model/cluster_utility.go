// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

const (
	// PrometheusOperatorCanonicalName is the canonical string representation of prometheus operator
	PrometheusOperatorCanonicalName = "prometheus-operator"
	// ThanosCanonicalName is the canonical string representation of thanos
	ThanosCanonicalName = "thanos"
	// NginxCanonicalName is the canonical string representation of nginx
	NginxCanonicalName = "nginx"
	// NginxInternalCanonicalName is the canonical string representation of nginx internal
	NginxInternalCanonicalName = "nginx-internal"
	// FluentbitCanonicalName is the canonical string representation of fluentbit
	FluentbitCanonicalName = "fluentbit"
	// TeleportCanonicalName is the canonical string representation of teleport
	TeleportCanonicalName = "teleport"
	// PgbouncerCanonicalName is the canonical string representation of pgbouncer
	PgbouncerCanonicalName = "pgbouncer"
	// PromtailCanonicalName is the canonical string representation of promtail
	PromtailCanonicalName = "promtail"
	// RtcdCanonicalName is the canonical string representation of RTCD
	RtcdCanonicalName = "rtcd"
	// NodeProblemDetectorCanonicalName is the canonical string representation of node problem detector
	NodeProblemDetectorCanonicalName = "node-problem-detector"
	// MetricsServerCanonicalName is the canonical string representation of metrics server
	MetricsServerCanonicalName = "metrics-server"
	// VeleroCanonicalName is the canonical string representation of velero
	VeleroCanonicalName = "velero"
	// CloudproberCanonicalName is the canonical string representation of Cloudprber
	CloudproberCanonicalName = "cloudprober"
	// UnmanagedUtilityVersion is the value of utility versions where the
	// provisioner will no longer manage the helm chart.
	UnmanagedUtilityVersion = "unmanaged"
	// GitlabOAuthTokenKey is the name of the Environment Variable which
	// may contain an OAuth token for accessing GitLab repositories over
	// HTTPS, used for fetching values files
	GitlabOAuthTokenKey = "GITLAB_OAUTH_TOKEN"
	// GitOpsRepoPath is the name of the Environment Variable which
	// contains the path to the gitops repo. e.g. /cloud/gitops.git
	GitOpsRepoPath = "GITOPS_REPO_PATH"
)

// gitlabToken is the token that will be used for remote helm charts.
var gitlabToken string

var gitOpsRepoURL string
var gitOpsRepoPath string

// SetGitlabToken is used to define the gitlab token that will be used for remote
// helm charts.
func SetGitlabToken(val string) {
	gitlabToken = val
}

// GetGitlabToken returns the value of gitlabToken.
func GetGitlabToken() string {
	return gitlabToken
}

func SetGitopsRepoURL(val string) {
	gitOpsRepoURL = val
}

func GetGitopsRepoURL() string {
	return gitOpsRepoURL
}

func SetGitopsRepoPath(val string) {
	gitOpsRepoPath = val
}

func GetGitopsRepoPath() string {
	return gitOpsRepoPath
}

// DefaultUtilityVersions holds the default values for all the HelmUtilityVersions
var DefaultUtilityVersions map[string]*HelmUtilityVersion = map[string]*HelmUtilityVersion{
	// PrometheusOperatorCanonicalName defines the default version and values path for the Helm chart
	PrometheusOperatorCanonicalName: {Chart: "54.2.2", ValuesPath: ""},
	// ThanosCanonicalName defines the default version and values path for the Helm chart
	ThanosCanonicalName: {Chart: "13.2.2", ValuesPath: ""},
	// NginxCanonicalName defines the default version and values path for the Helm chart
	NginxCanonicalName: {Chart: "4.11.1", ValuesPath: ""},
	// NginxInternalCanonicalName defines the default version and values path for the Helm chart
	NginxInternalCanonicalName: {Chart: "4.11.1", ValuesPath: ""},
	// FluentbitCanonicalName defines the default version and values path for the Helm chart
	FluentbitCanonicalName: {Chart: "0.31.0", ValuesPath: ""},
	// TeleportCanonicalName defines the default version and values path for the Helm chart
	TeleportCanonicalName: {Chart: "12.4.34", ValuesPath: ""},
	// PgbouncerCanonicalName defines the default version and values path for the Helm chart
	PgbouncerCanonicalName: {Chart: "1.2.1", ValuesPath: ""},
	// PromtailCanonicalName defines the default version and values path for the Helm chart
	PromtailCanonicalName: {Chart: "6.15.5", ValuesPath: ""},
	// RtcdCanonicalName defines the default version and values path for the Helm chart
	RtcdCanonicalName: {Chart: "1.3.0", ValuesPath: ""},
	// NodeProblemDetectorCanonicalName defines the default version and values path for the Helm chart
	NodeProblemDetectorCanonicalName: {Chart: "2.3.12", ValuesPath: ""},
	// MetricsServerCanonicalName defines the default version and values path for the Helm chart
	MetricsServerCanonicalName: {Chart: "3.12.1", ValuesPath: ""},
	// VeleroCanonicalName defines the default version for the Helm chart
	VeleroCanonicalName: {Chart: "5.1.5", ValuesPath: ""},
	// CloudproberCanonicalName defines the default version for the Helm chart
	CloudproberCanonicalName: {Chart: "0.1.4", ValuesPath: ""},
}

var defaultUtilityValuesFileNames map[string]string = map[string]string{
	PrometheusOperatorCanonicalName:  "prometheus_operator_values.yaml",
	ThanosCanonicalName:              "thanos_values.yaml",
	NginxCanonicalName:               "nginx_values.yaml",
	NginxInternalCanonicalName:       "nginx_internal_values.yaml",
	FluentbitCanonicalName:           "fluent-bit_values.yaml",
	TeleportCanonicalName:            "teleport_values.yaml",
	PgbouncerCanonicalName:           "pgbouncer_values.yaml",
	PromtailCanonicalName:            "promtail_values.yaml",
	RtcdCanonicalName:                "rtcd_values.yaml",
	NodeProblemDetectorCanonicalName: "node_problem_detector_values.yaml",
	MetricsServerCanonicalName:       "metrics_server_values.yaml",
	VeleroCanonicalName:              "velero_values.yaml",
	CloudproberCanonicalName:         "cloudprober_values.yaml",
}

var (
	// TODO make these configurable if the gitlab repo must ever be
	// moved, or if we ever need to specify a different branch or folder
	// (environment) name to pull the values files from
	gitlabProjectPath    string = "/api/v4/projects/%d/repository/files/%s" + `%%2F` + "%s?ref=%s"
	defaultProjectNumber int    = 295
	defaultEnvironment          = "dev"
	defaultBranch               = "master"
)

// SetUtilityDefaults is used to set Utility default version and values.
func SetUtilityDefaults(url string) {
	for utility, filename := range defaultUtilityValuesFileNames {
		if DefaultUtilityVersions[utility].ValuesPath == "" {
			DefaultUtilityVersions[utility].ValuesPath = fmt.Sprintf("%s%s", url, buildValuesPath(filename))
		}
	}
}

func buildValuesPath(filename string) string {
	return fmt.Sprintf(gitlabProjectPath,
		defaultProjectNumber,
		defaultEnvironment,
		filename,
		defaultBranch)
}

// UtilityGroupVersions holds the concrete metadata for any cluster
// utilities
type UtilityGroupVersions struct {
	PrometheusOperator  *HelmUtilityVersion
	Thanos              *HelmUtilityVersion
	Nginx               *HelmUtilityVersion
	NginxInternal       *HelmUtilityVersion
	Fluentbit           *HelmUtilityVersion
	Teleport            *HelmUtilityVersion
	Pgbouncer           *HelmUtilityVersion
	Promtail            *HelmUtilityVersion
	Rtcd                *HelmUtilityVersion
	NodeProblemDetector *HelmUtilityVersion
	MetricsServer       *HelmUtilityVersion
	Velero              *HelmUtilityVersion
	Cloudprober         *HelmUtilityVersion
}

type UtilityArgocdClusterRegister struct {
	ClusterType string `json:"ClusterType,omitempty"`
	Registered  bool   `json:"Registered,omitempty"`
}

// AsMap returns the UtilityGroupVersion represented as a map with the
// canonical names for each utility as the keys and the members of the
// struct making up the values
func (h *UtilityGroupVersions) AsMap() map[string]*HelmUtilityVersion {
	return map[string]*HelmUtilityVersion{
		PrometheusOperatorCanonicalName:  h.PrometheusOperator,
		ThanosCanonicalName:              h.Thanos,
		NginxCanonicalName:               h.Nginx,
		NginxInternalCanonicalName:       h.NginxInternal,
		FluentbitCanonicalName:           h.Fluentbit,
		TeleportCanonicalName:            h.Teleport,
		PgbouncerCanonicalName:           h.Pgbouncer,
		PromtailCanonicalName:            h.Promtail,
		RtcdCanonicalName:                h.Rtcd,
		NodeProblemDetectorCanonicalName: h.NodeProblemDetector,
		MetricsServerCanonicalName:       h.MetricsServer,
		VeleroCanonicalName:              h.Velero,
		CloudproberCanonicalName:         h.Cloudprober,
	}
}

// UtilityMetadata is a container struct for any metadata related to
// cluster utilities that needs to be persisted in the database
type UtilityMetadata struct {
	DesiredVersions       UtilityGroupVersions
	ActualVersions        UtilityGroupVersions
	ManagedByArgocd       bool
	ArgocdClusterRegister UtilityArgocdClusterRegister
}

// NewUtilityMetadata creates an instance of UtilityMetadata given the raw
// utility metadata.
func NewUtilityMetadata(metadataBytes []byte) (*UtilityMetadata, error) {
	// Check if length of metadata is 0 as opposed to if the value is nil. This
	// is done to avoid an issue encountered where the metadata value provided
	// had a length of 0, but had non-zero capacity.
	if len(metadataBytes) == 0 || string(metadataBytes) == "null" {
		return nil, nil
	}

	utilityMetadata := UtilityMetadata{}
	err := json.Unmarshal(metadataBytes, &utilityMetadata)
	if err != nil {
		return nil, err
	}

	return &utilityMetadata, nil
}

// SetUtilityActualVersion stores the provided version for the
// provided utility in the UtilityMetadata JSON []byte in this Cluster
func (c *Cluster) SetUtilityActualVersion(utility string, version *HelmUtilityVersion) error {
	metadata := &UtilityMetadata{}
	if c.UtilityMetadata != nil {
		metadata = c.UtilityMetadata
	}

	setUtilityVersion(&metadata.ActualVersions, utility, version)
	setUtilityVersion(&metadata.DesiredVersions, utility, nil)

	c.UtilityMetadata = metadata
	return nil
}

// SetUtilityDesiredVersions takes a map of string to string representing
// any metadata related to the utility group and stores it as a []byte
// in Cluster so that it can be inserted into the database
func (c *Cluster) SetUtilityDesiredVersions(desiredVersions map[string]*HelmUtilityVersion) {
	if c.UtilityMetadata == nil {
		c.UtilityMetadata = new(UtilityMetadata)
	}
	if desiredVersions == nil {
		desiredVersions = map[string]*HelmUtilityVersion{}
	}
	for utility, version := range desiredVersions {
		setUtilityVersion(&c.UtilityMetadata.DesiredVersions, utility, version)
	}
}

// DesiredUtilityVersion fetches the desired version of a utility from the
// Cluster object
func (c *Cluster) DesiredUtilityVersion(utility string) *HelmUtilityVersion {
	// some clusters may only be using pinned stable version, so an
	// empty UtilityMetadata field is possible; in this context it means
	// "utility"'s desired version is nothing
	if c.UtilityMetadata == nil {
		return nil
	}

	return getUtilityVersion(c.UtilityMetadata.DesiredVersions, utility)
}

// ActualUtilityVersion fetches the desired version of a utility from the
// Cluster object
func (c *Cluster) ActualUtilityVersion(utility string) *HelmUtilityVersion {
	if c.UtilityMetadata == nil {
		return nil
	}
	return getUtilityVersion(c.UtilityMetadata.ActualVersions, utility)
}

// UtilityMetadataFromReader produces a UtilityMetadata object from
// the JSON representation embedded in a io.Reader
func UtilityMetadataFromReader(reader io.Reader) (*UtilityMetadata, error) {
	utilityMetadata := UtilityMetadata{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&utilityMetadata)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &utilityMetadata, nil
}

// Gets the version for a utility from a utilityVersions struct using
// the utility's name's string representation for lookup
func getUtilityVersion(versions UtilityGroupVersions, utility string) *HelmUtilityVersion {
	if r, ok := versions.AsMap()[utility]; ok {
		return r
	}
	return nil
}

// setUtilityVersion will assign the version in desiredVersion to the
// utility whose name's string representation matches one of the known
// utilities with a version field in utilityVersion struct in the
// first argument
func setUtilityVersion(versions *UtilityGroupVersions, utility string, desiredVersion *HelmUtilityVersion) {
	switch utility {
	case PrometheusOperatorCanonicalName:
		versions.PrometheusOperator = desiredVersion
	case ThanosCanonicalName:
		versions.Thanos = desiredVersion
	case NginxCanonicalName:
		versions.Nginx = desiredVersion
	case NginxInternalCanonicalName:
		versions.NginxInternal = desiredVersion
	case FluentbitCanonicalName:
		versions.Fluentbit = desiredVersion
	case TeleportCanonicalName:
		versions.Teleport = desiredVersion
	case PgbouncerCanonicalName:
		versions.Pgbouncer = desiredVersion
	case PromtailCanonicalName:
		versions.Promtail = desiredVersion
	case RtcdCanonicalName:
		versions.Rtcd = desiredVersion
	case NodeProblemDetectorCanonicalName:
		versions.NodeProblemDetector = desiredVersion
	case MetricsServerCanonicalName:
		versions.MetricsServer = desiredVersion
	case VeleroCanonicalName:
		versions.Velero = desiredVersion
	case CloudproberCanonicalName:
		versions.Cloudprober = desiredVersion
	}
}

// HelmUtilityVersion holds the chart version and the values path.
type HelmUtilityVersion struct {
	Chart      string
	ValuesPath string
}

// UnmarshalJSON tries to unmarshal the HelmUtilityVersion from JSON
// If it fails, it assumes that bytes is just the chart version in string
func (u *HelmUtilityVersion) UnmarshalJSON(bytes []byte) error {
	type newHelmUtilityVersion HelmUtilityVersion
	err := json.Unmarshal(bytes, (*newHelmUtilityVersion)(u))
	if err != nil {
		u.Chart = strings.Trim(string(bytes), `"`)
	}
	return nil
}

// Version returns the Helm chart version
func (u *HelmUtilityVersion) Version() string {
	return u.Chart
}

// Values returns the name of the branch on which to find the correct
// values file
func (u *HelmUtilityVersion) Values() string {
	return u.ValuesPath
}

// IsEmpty returns true if the HelmUtilityVersion is nil or if either of the
// values inside are undefined. If HelmUtilityVersion is "unmanaged" then false
// is returned instead.
func (u *HelmUtilityVersion) IsEmpty() bool {
	if u == nil {
		return true
	}

	if u.Chart == UnmanagedUtilityVersion {
		return false
	}

	return u.ValuesPath == "" || u.Chart == ""
}

// UtilityIsUnmanaged returns true if the desired version of a utility is set to
// "unmanaged" or if the actual version is "unmanaged" and there is no new
// desired version.
func UtilityIsUnmanaged(desired *HelmUtilityVersion, actual *HelmUtilityVersion) bool {
	// Perform nil checks to derive version strings safely and help simplify the
	// logic later.
	var desiredVersion string
	if desired != nil {
		desiredVersion = desired.Version()
	}
	var actualVersion string
	if actual != nil {
		actualVersion = actual.Version()
	}

	if desiredVersion == UnmanagedUtilityVersion {
		return true
	}
	if desiredVersion == "" && actualVersion == UnmanagedUtilityVersion {
		return true
	}

	return false
}

func (c *Cluster) SetManagedByArgocd(clusterValues map[string]string) {
	if _, ok := clusterValues["force"]; ok {
		c.UtilityMetadata.ArgocdClusterRegister.Registered = false
	}
	if (len(clusterValues)) != 0 && (!c.UtilityMetadata.ArgocdClusterRegister.Registered) {
		for clusterValueKey, clusterValue := range clusterValues {
			setArgocdClusterValues(&c.UtilityMetadata.ArgocdClusterRegister, clusterValueKey, clusterValue)
		}
		c.UtilityMetadata.ManagedByArgocd = true
	}
}

func setArgocdClusterValues(clusterValues *UtilityArgocdClusterRegister, clusterValueKey, clusterValue string) {
	switch clusterValueKey {
	case "cluster-type":
		clusterValues.ClusterType = clusterValue
	}
}
