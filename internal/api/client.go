package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"

	"github.com/mattermost/mattermost-cloud/internal/model"
	"github.com/pkg/errors"
)

// Client is the programmatic interface to the provisioning server API.
type Client struct {
	address    string
	httpClient *http.Client
}

// NewClient creates a client to the provisioning server at the given address.
func NewClient(address string) *Client {
	return &Client{
		address:    address,
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
	return c.httpClient.Get(u)
}

func (c *Client) doPost(u string, request interface{}) (*http.Response, error) {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	return c.httpClient.Post(u, "application/json", bytes.NewReader(requestBytes))
}

func (c *Client) doPut(u string, request interface{}) (*http.Response, error) {
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal request")
	}

	httpRequest, err := http.NewRequest(http.MethodPut, u, bytes.NewReader(requestBytes))
	if err != nil {
		return nil, err
	}

	return c.httpClient.Do(httpRequest)
}

func (c *Client) doDelete(u string) (*http.Response, error) {
	request, err := http.NewRequest(http.MethodDelete, u, nil)
	if err != nil {
		return nil, err
	}

	return c.httpClient.Do(request)
}

// CreateCluster requests the creation of a cluster from the configured provisioning server.
func (c *Client) CreateCluster(request *CreateClusterRequest) (*model.Cluster, error) {
	resp, err := c.doPost(c.buildURL("/api/clusters"), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return model.ClusterFromReader(resp.Body)

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

// GetCluster fetches the specified cluster from the configured provisioning server.
func (c *Client) GetCluster(clusterID string) (*model.Cluster, error) {
	resp, err := c.doGet(c.buildURL("/api/cluster/%s", clusterID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return model.ClusterFromReader(resp.Body)

	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetClusters fetches the list of clusters from the configured provisioning server.
func (c *Client) GetClusters(request *GetClustersRequest) ([]*model.Cluster, error) {
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
		return model.ClustersFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// UpgradeCluster upgrades a cluster to the latest recommended production ready k8s version.
func (c *Client) UpgradeCluster(clusterID, version string) error {
	resp, err := c.doPut(c.buildURL("/api/cluster/%s/kubernetes/%s", clusterID, version), nil)
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

// CreateInstallation requests the creation of a installation from the configured provisioning server.
func (c *Client) CreateInstallation(request *CreateInstallationRequest) (*model.Installation, error) {
	resp, err := c.doPost(c.buildURL("/api/installations"), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusAccepted:
		return model.InstallationFromReader(resp.Body)

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
func (c *Client) GetInstallation(installationID string) (*model.Installation, error) {
	resp, err := c.doGet(c.buildURL("/api/installation/%s", installationID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return model.InstallationFromReader(resp.Body)

	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetInstallations fetches the list of installations from the configured provisioning server.
func (c *Client) GetInstallations(request *GetInstallationsRequest) ([]*model.Installation, error) {
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
		return model.InstallationsFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// UpgradeInstallation upgrades a installation to the given Mattermost version.
func (c *Client) UpgradeInstallation(installationID, version string) error {
	resp, err := c.doPut(c.buildURL("/api/installation/%s/mattermost/%s", installationID, version), nil)
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

// GetClusterInstallation fetches the specified cluster installation from the configured provisioning server.
func (c *Client) GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error) {
	resp, err := c.doGet(c.buildURL("/api/cluster_installation/%s", clusterInstallationID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return model.ClusterInstallationFromReader(resp.Body)

	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetClusterInstallations fetches the list of cluster installations from the configured provisioning server.
func (c *Client) GetClusterInstallations(request *GetClusterInstallationsRequest) ([]*model.ClusterInstallation, error) {
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
		return model.ClusterInstallationsFromReader(resp.Body)

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
		return model.ClusterInstallationConfigFromReader(resp.Body)

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

// CreateGroup requests the creation of a group from the configured provisioning server.
func (c *Client) CreateGroup(request *CreateGroupRequest) (*model.Group, error) {
	resp, err := c.doPost(c.buildURL("/api/groups"), request)
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return model.GroupFromReader(resp.Body)

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// UpdateGroup updates the installation group.
func (c *Client) UpdateGroup(request *PatchGroupRequest) error {
	resp, err := c.doPut(c.buildURL("/api/group/%s", request.ID), request)
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
func (c *Client) GetGroup(groupID string) (*model.Group, error) {
	resp, err := c.doGet(c.buildURL("/api/group/%s", groupID))
	if err != nil {
		return nil, err
	}
	defer closeBody(resp)

	switch resp.StatusCode {
	case http.StatusOK:
		return model.GroupFromReader(resp.Body)

	case http.StatusNotFound:
		return nil, nil

	default:
		return nil, errors.Errorf("failed with status code %d", resp.StatusCode)
	}
}

// GetGroups fetches the list of groups from the configured provisioning server.
func (c *Client) GetGroups(request *GetGroupsRequest) ([]*model.Group, error) {
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
		return model.GroupsFromReader(resp.Body)

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
func (c *Client) LeaveGroup(installationID string) error {
	resp, err := c.doDelete(c.buildURL("/api/installation/%s/group", installationID))
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
