package api

import (
	"github.com/mattermost/mattermost-cloud/internal/model"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/sirupsen/logrus"
)

// Provisioner describes the interface for provisioning clusters required by the API context.
type Provisioner interface {
	CreateCluster(request *CreateClusterRequest) (*model.Cluster, error)
	UpgradeCluster(clusterID string, version string) error
	DeleteCluster(clusterID string) error
}

// Context provides the API with all necessary data and interfaces for responding to requests.
//
// It is cloned before each request, allowing per-request changes such as logger annotations.
type Context struct {
	SQLStore    *store.SQLStore
	Provisioner Provisioner
	Logger      logrus.FieldLogger
}

// Clone creates a shallow copy of context, allowing clones to apply per-request changes.
func (c *Context) Clone() *Context {
	return &Context{
		SQLStore:    c.SQLStore,
		Provisioner: c.Provisioner,
		Logger:      c.Logger,
	}
}
