// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"fmt"
	"io"
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
	// StackroxCanonicalName is the canonical string representation of stackrox
	StackroxCanonicalName = "stackrox-secured-cluster-services"
	// KubecostCanonicalName is the canonical string representation of kubecost
	KubecostCanonicalName = "kubecost"
	// GitlabOAuthTokenKey is the name of the Environment Variable which
	// may contain an OAuth token for accessing GitLab repositories over
	// HTTPS, used for fetching values files
	GitlabOAuthTokenKey = "GITLAB_OAUTH_TOKEN"
)

// DefaultUtilityVersions holds the default values for all of the HelmUtilityVersions
var DefaultUtilityVersions map[string]*HelmUtilityVersion = map[string]*HelmUtilityVersion{
	// PrometheusOperatorDefaultVersion defines the default version for the Helm chart
	PrometheusOperatorCanonicalName: {Chart: "9.4.4", ValuesPath: ""},
	// ThanosDefaultVersion defines the default version for the Helm chart
	ThanosCanonicalName: {Chart: "3.2.2", ValuesPath: ""},
	// NginxDefaultVersion defines the default version for the Helm chart
	NginxCanonicalName: {Chart: "2.15.0", ValuesPath: ""},
	// NginxInternalDefaultVersion defines the default version for the Helm chart
	NginxInternalCanonicalName: {Chart: "2.15.0", ValuesPath: ""},
	// FluentbitDefaultVersion defines the default version for the Helm chart
	FluentbitCanonicalName: {Chart: "0.15.8", ValuesPath: ""},
	// TeleportDefaultVersion defines the default version for the Helm chart
	TeleportCanonicalName: {Chart: "0.3.0", ValuesPath: ""},
	// PgbouncerDefaultVersion defines the default version for the Helm chart
	PgbouncerCanonicalName: {Chart: "1.1.0", ValuesPath: ""},
	// StackroxDefaultVersion defines the default version for the Helm chart
	StackroxCanonicalName: {Chart: "62.0.0", ValuesPath: ""},
	// KubecostDefaultVersion defines the default version for the Helm chart
	KubecostCanonicalName: {Chart: "1.83.1", ValuesPath: ""},
}

// SetUtilityDefaults is used to set Utility default version and values.
func SetUtilityDefaults(url string) {
	if DefaultUtilityVersions[PrometheusOperatorCanonicalName].ValuesPath == "" {
		DefaultUtilityVersions[PrometheusOperatorCanonicalName].ValuesPath = fmt.Sprintf("%s/api/v4/projects/33/repository/files/dev%%2Fprometheus_operator_values.yaml?ref=master", url)
	}
	if DefaultUtilityVersions[ThanosCanonicalName].ValuesPath == "" {
		DefaultUtilityVersions[ThanosCanonicalName].ValuesPath = fmt.Sprintf("%s/api/v4/projects/33/repository/files/dev%%2Fthanos_values.yaml?ref=master", url)
	}
	if DefaultUtilityVersions[NginxCanonicalName].ValuesPath == "" {
		DefaultUtilityVersions[NginxCanonicalName].ValuesPath = fmt.Sprintf("%s/api/v4/projects/33/repository/files/dev%%2Fnginx_values.yaml?ref=master", url)
	}
	if DefaultUtilityVersions[NginxInternalCanonicalName].ValuesPath == "" {
		DefaultUtilityVersions[NginxInternalCanonicalName].ValuesPath = fmt.Sprintf("%s/api/v4/projects/33/repository/files/dev%%2Fnginx_internal_values.yaml?ref=master", url)
	}
	if DefaultUtilityVersions[FluentbitCanonicalName].ValuesPath == "" {
		DefaultUtilityVersions[FluentbitCanonicalName].ValuesPath = fmt.Sprintf("%s/api/v4/projects/33/repository/files/dev%%2Ffluent-bit_values.yaml?ref=master", url)
	}
	if DefaultUtilityVersions[TeleportCanonicalName].ValuesPath == "" {
		DefaultUtilityVersions[TeleportCanonicalName].ValuesPath = fmt.Sprintf("%s/api/v4/projects/33/repository/files/dev%%2Fteleport_values.yaml?ref=master", url)
	}
	if DefaultUtilityVersions[PgbouncerCanonicalName].ValuesPath == "" {
		DefaultUtilityVersions[PgbouncerCanonicalName].ValuesPath = fmt.Sprintf("%s/api/v4/projects/33/repository/files/dev%%2Fpgbouncer_values.yaml?ref=master", url)
	}
	if DefaultUtilityVersions[StackroxCanonicalName].ValuesPath == "" {
		DefaultUtilityVersions[StackroxCanonicalName].ValuesPath = fmt.Sprintf("%s/api/v4/projects/33/repository/files/dev%%2Fstackrox_values.yaml?ref=master", url)
	}
	if DefaultUtilityVersions[KubecostCanonicalName].ValuesPath == "" {
		DefaultUtilityVersions[KubecostCanonicalName].ValuesPath = fmt.Sprintf("%s/api/v4/projects/33/repository/files/dev%%2Fkubecost_values.yaml?ref=master", url)
	}
}

// UtilityGroupVersions holds the concrete metadata for any cluster
// utilities
type UtilityGroupVersions struct {
	PrometheusOperator *HelmUtilityVersion
	Thanos             *HelmUtilityVersion
	Nginx              *HelmUtilityVersion
	NginxInternal      *HelmUtilityVersion
	Fluentbit          *HelmUtilityVersion
	Teleport           *HelmUtilityVersion
	Pgbouncer          *HelmUtilityVersion
	Stackrox           *HelmUtilityVersion
	Kubecost           *HelmUtilityVersion
}

// AsMap returns the UtilityGroupVersion represented as a map with the
// canonical names for each utility as the keys and the members of the
// struct making up the values
func (h *UtilityGroupVersions) AsMap() map[string]*HelmUtilityVersion {
	return map[string]*HelmUtilityVersion{
		PrometheusOperatorCanonicalName: h.PrometheusOperator,
		ThanosCanonicalName:             h.Thanos,
		NginxCanonicalName:              h.Nginx,
		NginxInternalCanonicalName:      h.NginxInternal,
		FluentbitCanonicalName:          h.Fluentbit,
		TeleportCanonicalName:           h.Teleport,
		PgbouncerCanonicalName:          h.Pgbouncer,
		StackroxCanonicalName:           h.Stackrox,
		KubecostCanonicalName:           h.Kubecost,
	}
}

// UtilityMetadata is a container struct for any metadata related to
// cluster utilities that needs to be persisted in the database
type UtilityMetadata struct {
	DesiredVersions UtilityGroupVersions
	ActualVersions  UtilityGroupVersions
}

// NewUtilityMetadata creates an instance of UtilityMetadata given the raw
// utility metadata.
func NewUtilityMetadata(metadataBytes []byte) (*UtilityMetadata, error) {
	// Check if length of metadata is 0 as opposed to if the value is nil. This
	// is done to avoid an issue encountered where the metadata value provided
	// had a length of 0, but had non-zero capacity.
	if len(metadataBytes) == 0 || string(metadataBytes) == "null" {
		// TODO: remove "null" check after sqlite is gone.
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
func (c *Cluster) SetUtilityDesiredVersions(versions map[string]*HelmUtilityVersion) error {
	desiredVersions := make(map[string]*HelmUtilityVersion)
	if c.UtilityMetadata == nil {
		c.UtilityMetadata = new(UtilityMetadata)
	}
	// at create time there will be no actual versions and it's ok
	// create will have defaults
	for k, v := range c.UtilityMetadata.ActualVersions.AsMap() {
		desiredVersions[k] = v
	}

	for utility, version := range versions {
		desiredVersions[utility] = version
	}

	for utility, version := range desiredVersions {
		setUtilityVersion(&c.UtilityMetadata.DesiredVersions, utility, version)
	}

	return nil
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
	case StackroxCanonicalName:
		versions.Stackrox = desiredVersion
	case KubecostCanonicalName:
		versions.Kubecost = desiredVersion
	}
}

// HelmUtilityVersion holds the chart version and the version of the
// values file
type HelmUtilityVersion struct {
	Chart      string
	ValuesPath string
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
