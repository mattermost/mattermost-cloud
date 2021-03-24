// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// Client is the programmatic interface to the provisioning server API.
type Client struct {
	address    string
	headers    map[string]string
	httpClient *http.Client
}

// NewClient creates a client to the provisioning server at the given address.
func NewClient(address string) *Client {
	return &Client{
		address:    address,
		headers:    make(map[string]string),
		httpClient: &http.Client{},
	}
}

// NewClientWithHeaders creates a client to the provisioning server at the given
// address and uses the provided headers.
func NewClientWithHeaders(address string, headers map[string]string) *Client {
	return &Client{
		address:    address,
		headers:    headers,
		httpClient: &http.Client{},
	}
}

// closeBody ensures the Body of an http.Response is properly closed.
func closeBody(r *http.Response) {
	if r.Body != nil {
		_, _ = ioutil.ReadAll(r.Body)
		_ = r.Body.Close()
	}
}

func (c *Client) buildURL(urlPath string, args ...interface{}) string {
	return fmt.Sprintf("%s%s", c.address, fmt.Sprintf(urlPath, args...))
}

func (c *Client) doGet(u string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	for k, v := range c.headers {
		req.Header.Add(k, v)
	}

	return c.httpClient.Do(req)
}

func (c *Client) doPost(u string, request interface{}) (*http.Response, error) {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	req, err := http.NewRequest(http.MethodPost, u, bytes.NewReader(requestBytes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	for k, v := range c.headers {
		req.Header.Add(k, v)
	}
	req.Header.Set("Content-Type", "application/json")

	return c.httpClient.Do(req)
}

func (c *Client) doPut(u string, request interface{}) (*http.Response, error) {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	req, err := http.NewRequest(http.MethodPut, u, bytes.NewReader(requestBytes))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	for k, v := range c.headers {
		req.Header.Add(k, v)
	}

	return c.httpClient.Do(req)
}

func (c *Client) doDelete(u string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create http request")
	}
	for k, v := range c.headers {
		req.Header.Add(k, v)
	}

	return c.httpClient.Do(req)
}

// CreateCluster requests the creation of a cluster from the configured provisioning server.
func (c *Client) CreateCluster(request *CreateClusterRequest) (*ClusterDTO, error) {
	resp, err := c.doPost(c.buildURL("/api/clusters"), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return ClusterDTOFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// RetryCreateCluster retries the creation of a cluster from the configured provisioning server.
func (c *Client) RetryCreateCluster(clusterID string) error {
	resp, err := c.doPost(c.buildURL("/api/cluster/%s", clusterID), nil)
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// ProvisionCluster provisions k8s operators and Helm charts on a
// cluster from the configured provisioning server.
func (c *Client) ProvisionCluster(clusterID string, request *ProvisionClusterRequest) (*ClusterDTO, error) {
	resp, err := c.doPost(c.buildURL("/api/cluster/%s/provision", clusterID), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return ClusterDTOFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetCluster fetches the specified cluster from the configured provisioning server.
func (c *Client) GetCluster(clusterID string) (*ClusterDTO, error) {
	resp, err := c.doGet(c.buildURL("/api/cluster/%s", clusterID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return ClusterDTOFromReader(resp.Body)

	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetClusters fetches the list of clusters from the configured provisioning server.
func (c *Client) GetClusters(request *GetClustersRequest) ([]*ClusterDTO, error) {
	u, err := url.Parse(c.buildURL("/api/clusters"))
	if err != nil {
		return nil, err
	}

	request.ApplyToURL(u)

	resp, err := c.doGet(u.String())
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return ClusterDTOsFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetClusterUtilities returns the metadata for all utilities running in the given cluster.
func (c *Client) GetClusterUtilities(clusterID string) (*UtilityMetadata, error) {
	resp, err := c.doGet(c.buildURL("/api/cluster/%s/utilities", clusterID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return UtilityMetadataFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// UpdateCluster updates a cluster's configuration.
func (c *Client) UpdateCluster(clusterID string, request *UpdateClusterRequest) (*ClusterDTO, error) {
	resp, err := c.doPut(c.buildURL("/api/cluster/%s", clusterID), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return ClusterDTOFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// UpgradeCluster upgrades a cluster to the latest recommended production ready k8s version.
func (c *Client) UpgradeCluster(clusterID string, request *PatchUpgradeClusterRequest) (*ClusterDTO, error) {
	resp, err := c.doPut(c.buildURL("/api/cluster/%s/kubernetes", clusterID), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return ClusterDTOFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// ResizeCluster resizes a cluster with a new size value.
func (c *Client) ResizeCluster(clusterID string, request *PatchClusterSizeRequest) (*ClusterDTO, error) {
	resp, err := c.doPut(c.buildURL("/api/cluster/%s/size", clusterID), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return ClusterDTOFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// DeleteCluster deletes the given cluster and all resources contained therein.
func (c *Client) DeleteCluster(clusterID string) error {
	resp, err := c.doDelete(c.buildURL("/api/cluster/%s", clusterID))
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// AddClusterAnnotations adds annotations to the given cluster.
func (c *Client) AddClusterAnnotations(clusterID string, annotationsRequest *AddAnnotationsRequest) (*ClusterDTO, error) {
	resp, err := c.doPost(c.buildURL("/api/cluster/%s/annotations", clusterID), annotationsRequest)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return ClusterDTOFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// DeleteClusterAnnotation deletes annotation from the given cluster.
func (c *Client) DeleteClusterAnnotation(clusterID string, annotationName string) error {
	resp, err := c.doDelete(c.buildURL("/api/cluster/%s/annotation/%s", clusterID, annotationName))
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// CreateInstallation requests the creation of a installation from the configured provisioning server.
func (c *Client) CreateInstallation(request *CreateInstallationRequest) (*InstallationDTO, error) {
	resp, err := c.doPost(c.buildURL("/api/installations"), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return InstallationDTOFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// RetryCreateInstallation retries the creation of a installation from the configured provisioning server.
func (c *Client) RetryCreateInstallation(installationID string) error {
	resp, err := c.doPost(c.buildURL("/api/installation/%s", installationID), nil)
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetInstallation fetches the specified installation from the configured provisioning server.
func (c *Client) GetInstallation(installationID string, request *GetInstallationRequest) (*InstallationDTO, error) {
	u, err := url.Parse(c.buildURL("/api/installation/%s", installationID))
	if err != nil {
		return nil, err
	}

	request.ApplyToURL(u)

	resp, err := c.doGet(u.String())
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return InstallationDTOFromReader(resp.Body)

	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetInstallationByDNS finds an installation with the given FQDN.
func (c *Client) GetInstallationByDNS(DNS string, request *GetInstallationRequest) (*InstallationDTO, error) {
	if request == nil {
		request = &GetInstallationRequest{
			IncludeGroupConfig:          false,
			IncludeGroupConfigOverrides: false,
		}
	}
	installations, err := c.GetInstallations(&GetInstallationsRequest{
		IncludeGroupConfig:          request.IncludeGroupConfig,
		IncludeGroupConfigOverrides: request.IncludeGroupConfigOverrides,
		Paging:                      AllPagesNotDeleted(),
		DNS:                         DNS,
	})
	if err != nil {
		return nil, errors.Wrap(err, "problem getting installation")
	}

	if len(installations) > 1 {
		return nil, errors.Errorf("received ambiguous response (%d Installations) when expecting only one",
			len(installations))
	} else if len(installations) == 0 {
		return nil, nil
	}
	return installations[0], nil
}

// GetInstallations fetches the list of installations from the configured provisioning server.
func (c *Client) GetInstallations(request *GetInstallationsRequest) ([]*InstallationDTO, error) {
	u, err := url.Parse(c.buildURL("/api/installations"))
	if err != nil {
		return nil, err
	}

	request.ApplyToURL(u)

	resp, err := c.doGet(u.String())
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return InstallationDTOsFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetInstallationsCount returns then number of installations filtered by deleted field
func (c *Client) GetInstallationsCount(includeDeleted bool) (int64, error) {
	u, err := url.Parse(c.buildURL("/api/installations/count"))
	if err != nil {
		return 0, err
	}
	if includeDeleted {
		q := u.Query()
		q.Add("include_deleted", "true")
		u.RawQuery = q.Encode()
	}
	resp, err := c.doGet(u.String())
	if err != nil {
		return 0, errors.Wrap(err, "problem getting installations count")
	}
	defer closeBody(resp)
	switch resp.StatusCode {
	case http.StatusOK:
		return InstallationsCountFromReader(resp.Body)
	default:
		return 0, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// UpdateInstallation updates an installation.
func (c *Client) UpdateInstallation(installationID string, request *PatchInstallationRequest) (*InstallationDTO, error) {
	resp, err := c.doPut(c.buildURL("/api/installation/%s/mattermost", installationID), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return InstallationDTOFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// HibernateInstallation puts an installation into hibernation.
func (c *Client) HibernateInstallation(installationID string) (*InstallationDTO, error) {
	resp, err := c.doPost(c.buildURL("/api/installation/%s/hibernate", installationID), nil)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return InstallationDTOFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// WakeupInstallation wakes an installation from hibernation.
func (c *Client) WakeupInstallation(installationID string) (*InstallationDTO, error) {
	resp, err := c.doPost(c.buildURL("/api/installation/%s/wakeup", installationID), nil)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return InstallationDTOFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetInstallationsStatus fetches the status for all installations.
func (c *Client) GetInstallationsStatus() (*InstallationsStatus, error) {
	resp, err := c.doGet(c.buildURL("/api/installations/status"))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return InstallationsStatusFromReader(resp.Body)

	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// DeleteInstallation deletes the given installation and all resources contained therein.
func (c *Client) DeleteInstallation(installationID string) error {
	resp, err := c.doDelete(c.buildURL("/api/installation/%s", installationID))
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// AddInstallationAnnotations adds annotations to the given installation.
func (c *Client) AddInstallationAnnotations(installationID string, annotationsRequest *AddAnnotationsRequest) (*InstallationDTO, error) {
	resp, err := c.doPost(c.buildURL("/api/installation/%s/annotations", installationID), annotationsRequest)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return InstallationDTOFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// DeleteInstallationAnnotation deletes annotation from the given installation.
func (c *Client) DeleteInstallationAnnotation(installationID string, annotationName string) error {
	resp, err := c.doDelete(c.buildURL("/api/installation/%s/annotation/%s", installationID, annotationName))
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusNoContent:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// CreateInstallationBackup triggers backup for the given installation.
func (c *Client) CreateInstallationBackup(installationID string) (*InstallationBackup, error) {
	resp, err := c.doPost(c.buildURL("/api/installations/backups"), &InstallationBackupRequest{InstallationID: installationID})
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return NewInstallationBackupFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetInstallationBackups returns list of installation backups.
func (c *Client) GetInstallationBackups(request *GetInstallationBackupsRequest) ([]*InstallationBackup, error) {
	u, err := url.Parse(c.buildURL("/api/installations/backups"))
	if err != nil {
		return nil, err
	}

	request.ApplyToURL(u)

	resp, err := c.doGet(u.String())
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return NewInstallationBackupsFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetInstallationBackup returns given installation backup.
func (c *Client) GetInstallationBackup(backupID string) (*InstallationBackup, error) {
	resp, err := c.doGet(c.buildURL("/api/installations/backup/%s", backupID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return NewInstallationBackupFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// DeleteInstallationBackup deletes given installation backup.
func (c *Client) DeleteInstallationBackup(backupID string) error {
	resp, err := c.doDelete(c.buildURL("/api/installations/backup/%s", backupID))
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetClusterInstallation fetches the specified cluster installation from the configured provisioning server.
func (c *Client) GetClusterInstallation(clusterInstallationID string) (*ClusterInstallation, error) {
	resp, err := c.doGet(c.buildURL("/api/cluster_installation/%s", clusterInstallationID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return ClusterInstallationFromReader(resp.Body)

	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetClusterInstallations fetches the list of cluster installations from the configured provisioning server.
func (c *Client) GetClusterInstallations(request *GetClusterInstallationsRequest) ([]*ClusterInstallation, error) {
	u, err := url.Parse(c.buildURL("/api/cluster_installations"))
	if err != nil {
		return nil, err
	}

	request.ApplyToURL(u)

	resp, err := c.doGet(u.String())
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return ClusterInstallationsFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetClusterInstallationConfig fetches the specified cluster installation's Mattermost config.
func (c *Client) GetClusterInstallationConfig(clusterInstallationID string) (map[string]interface{}, error) {
	resp, err := c.doGet(c.buildURL("/api/cluster_installation/%s/config", clusterInstallationID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return ClusterInstallationConfigFromReader(resp.Body)

	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// SetClusterInstallationConfig modifies an cluster installation's Mattermost configuration.
//
// The operation is applied as a patch, preserving existing values if they are not specified.
func (c *Client) SetClusterInstallationConfig(clusterInstallationID string, config map[string]interface{}) error {
	resp, err := c.doPut(c.buildURL("/api/cluster_installation/%s/config", clusterInstallationID), config)
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// RunMattermostCLICommandOnClusterInstallation runs a Mattermost CLI command against a cluster installation.
func (c *Client) RunMattermostCLICommandOnClusterInstallation(clusterInstallationID string, subcommand []string) ([]byte, error) {
	resp, err := c.doPost(c.buildURL("/api/cluster_installation/%s/mattermost_cli", clusterInstallationID), subcommand)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	bytes, _ := ioutil.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		return bytes, nil

	default:
		return bytes, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// ExecClusterInstallationCLI runs a valid exec command against a cluster installation.
func (c *Client) ExecClusterInstallationCLI(clusterInstallationID, command string, subcommand []string) ([]byte, error) {
	resp, err := c.doPost(c.buildURL("/api/cluster_installation/%s/exec/%s", clusterInstallationID, command), subcommand)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	bytes, _ := ioutil.ReadAll(resp.Body)

	switch resp.StatusCode {
	case http.StatusOK:
		return bytes, nil

	default:
		return bytes, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// CreateGroup requests the creation of a group from the configured provisioning server.
func (c *Client) CreateGroup(request *CreateGroupRequest) (*Group, error) {
	resp, err := c.doPost(c.buildURL("/api/groups"), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return GroupFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// UpdateGroup updates the installation group.
func (c *Client) UpdateGroup(request *PatchGroupRequest) (*Group, error) {
	resp, err := c.doPut(c.buildURL("/api/group/%s", request.ID), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return GroupFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// DeleteGroup deletes the given group and all resources contained therein.
func (c *Client) DeleteGroup(groupID string) error {
	resp, err := c.doDelete(c.buildURL("/api/group/%s", groupID))
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetGroup fetches the specified group from the configured provisioning server.
func (c *Client) GetGroup(groupID string) (*Group, error) {
	resp, err := c.doGet(c.buildURL("/api/group/%s", groupID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return GroupFromReader(resp.Body)

	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetGroups fetches the list of groups from the configured provisioning server.
func (c *Client) GetGroups(request *GetGroupsRequest) ([]*Group, error) {
	u, err := url.Parse(c.buildURL("/api/groups"))
	if err != nil {
		return nil, err
	}

	request.ApplyToURL(u)

	resp, err := c.doGet(u.String())
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return GroupsFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetGroupStatus fetches the status for specified group from the configured provisioning server.
func (c *Client) GetGroupStatus(groupID string) (*GroupStatus, error) {
	resp, err := c.doGet(c.buildURL("/api/group/%s/status", groupID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return GroupStatusFromReader(resp.Body)

	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetGroupsStatus fetches the status for all groups.
func (c *Client) GetGroupsStatus() ([]*GroupsStatus, error) {
	resp, err := c.doGet(c.buildURL("/api/groups/status"))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return GroupsStatusFromReader(resp.Body)

	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// JoinGroup joins an installation to the given group, leaving any existing group.
func (c *Client) JoinGroup(groupID, installationID string) error {
	resp, err := c.doPut(c.buildURL("/api/installation/%s/group/%s", installationID, groupID), nil)
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// LeaveGroup removes an installation from its group, if any.
func (c *Client) LeaveGroup(installationID string, request *LeaveGroupRequest) error {
	u, err := url.Parse(c.buildURL("/api/installation/%s/group", installationID))
	if err != nil {
		return err
	}

	request.ApplyToURL(u)

	resp, err := c.doDelete(u.String())
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetMultitenantDatabases fetches the list of multitenant databases from the configured provisioning server.
func (c *Client) GetMultitenantDatabases(request *GetDatabasesRequest) ([]*MultitenantDatabase, error) {
	u, err := url.Parse(c.buildURL("/api/databases"))
	if err != nil {
		return nil, err
	}

	request.ApplyToURL(u)

	resp, err := c.doGet(u.String())
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return MultitenantDatabasesFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// CreateWebhook requests the creation of a webhook from the configured provisioning server.
func (c *Client) CreateWebhook(request *CreateWebhookRequest) (*Webhook, error) {
	resp, err := c.doPost(c.buildURL("/api/webhooks"), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return WebhookFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetWebhook fetches the webhook from the configured provisioning server.
func (c *Client) GetWebhook(webhookID string) (*Webhook, error) {
	resp, err := c.doGet(c.buildURL("/api/webhook/%s", webhookID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return WebhookFromReader(resp.Body)

	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetWebhooks fetches the list of webhooks from the configured provisioning server.
func (c *Client) GetWebhooks(request *GetWebhooksRequest) ([]*Webhook, error) {
	u, err := url.Parse(c.buildURL("/api/webhooks"))
	if err != nil {
		return nil, err
	}

	request.ApplyToURL(u)

	resp, err := c.doGet(u.String())
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return WebhooksFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// DeleteWebhook deletes the given webhook.
func (c *Client) DeleteWebhook(webhookID string) error {
	resp, err := c.doDelete(c.buildURL("/api/webhook/%s", webhookID))
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// LockAPIForCluster locks API changes for a given cluster.
func (c *Client) LockAPIForCluster(clusterID string) error {
	return c.makeSecurityCall("cluster", clusterID, "api", "lock")
}

// UnlockAPIForCluster unlocks API changes for a given cluster.
func (c *Client) UnlockAPIForCluster(clusterID string) error {
	return c.makeSecurityCall("cluster", clusterID, "api", "unlock")
}

// LockAPIForInstallation locks API changes for a given installation.
func (c *Client) LockAPIForInstallation(installationID string) error {
	return c.makeSecurityCall("installation", installationID, "api", "lock")
}

// UnlockAPIForInstallation unlocks API changes for a given installation.
func (c *Client) UnlockAPIForInstallation(installationID string) error {
	return c.makeSecurityCall("installation", installationID, "api", "unlock")
}

// LockAPIForClusterInstallation locks API changes for a given cluster installation.
func (c *Client) LockAPIForClusterInstallation(clusterID string) error {
	return c.makeSecurityCall("cluster_installation", clusterID, "api", "lock")
}

// UnlockAPIForClusterInstallation unlocks API changes for a given cluster installation.
func (c *Client) UnlockAPIForClusterInstallation(clusterID string) error {
	return c.makeSecurityCall("cluster_installation", clusterID, "api", "unlock")
}

// LockAPIForGroup locks API changes for a given group.
func (c *Client) LockAPIForGroup(groupID string) error {
	return c.makeSecurityCall("group", groupID, "api", "lock")
}

// UnlockAPIForGroup unlocks API changes for a given group.
func (c *Client) UnlockAPIForGroup(groupID string) error {
	return c.makeSecurityCall("group", groupID, "api", "unlock")
}

// LockAPIForBackup locks API changes for a given backup.
func (c *Client) LockAPIForBackup(backupID string) error {
	return c.makeSecurityCall("installation/backup", backupID, "api", "lock")
}

// UnlockAPIForBackup unlocks API changes for a given backup.
func (c *Client) UnlockAPIForBackup(backupID string) error {
	return c.makeSecurityCall("installation/backup", backupID, "api", "unlock")
}

func (c *Client) makeSecurityCall(resourceType, id, securityType, action string) error {
	resp, err := c.doPost(c.buildURL("/api/security/%s/%s/%s/%s", resourceType, id, securityType, action), nil)
	if err != nil {
		return err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return nil

	default:
		return errors.Errorf("failed with status code %d", resp.StatusCode)
	}

}
