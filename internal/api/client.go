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
