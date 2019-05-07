package api

import (
	"github.com/mattermost/mattermost-cloud/internal/model"
	"github.com/sirupsen/logrus"
)

// Supervisor describes the interface to notify the background jobs of an actionable change.
type Supervisor interface {
	Do() error
}

// Store describes the interface required to persist changes made via API requests.
type Store interface {
	CreateCluster(cluster *model.Cluster) error
	GetCluster(clusterID string) (*model.Cluster, error)
	GetClusters(filter *model.ClusterFilter) ([]*model.Cluster, error)
	GetUnlockedClusterPendingWork() (*model.Cluster, error)
	UpdateCluster(cluster *model.Cluster) error
	LockCluster(clusterID, lockerID string) (bool, error)
	UnlockCluster(clusterID, lockerID string, force bool) (bool, error)
	DeleteCluster(clusterID string) error
}

// Context provides the API with all necessary data and interfaces for responding to requests.
//
// It is cloned before each request, allowing per-request changes such as logger annotations.
type Context struct {
	Store      Store
	Supervisor Supervisor
	RequestID  string
	Logger     logrus.FieldLogger
}

// Clone creates a shallow copy of context, allowing clones to apply per-request changes.
func (c *Context) Clone() *Context {
	return &Context{
		Store:      c.Store,
		Supervisor: c.Supervisor,
		Logger:     c.Logger,
	}
}
