package api

import (
	"github.com/mattermost/mattermost-cloud/model"
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
	UpdateCluster(cluster *model.Cluster) error
	LockCluster(clusterID, lockerID string) (bool, error)
	UnlockCluster(clusterID, lockerID string, force bool) (bool, error)
	DeleteCluster(clusterID string) error

	CreateInstallation(installation *model.Installation) error
	GetInstallation(installationID string) (*model.Installation, error)
	GetInstallations(filter *model.InstallationFilter) ([]*model.Installation, error)
	UpdateInstallation(installation *model.Installation) error
	LockInstallation(installationID, lockerID string) (bool, error)
	UnlockInstallation(installationID, lockerID string, force bool) (bool, error)
	DeleteInstallation(installationID string) error

	GetClusterInstallation(clusterInstallationID string) (*model.ClusterInstallation, error)
	GetClusterInstallations(filter *model.ClusterInstallationFilter) ([]*model.ClusterInstallation, error)

	CreateGroup(group *model.Group) error
	GetGroup(groupID string) (*model.Group, error)
	GetGroups(filter *model.GroupFilter) ([]*model.Group, error)
	UpdateGroup(group *model.Group) error
	DeleteGroup(groupID string) error
}

// Provisioner describes the interface required to communicate with the Kubernetes cluster.
type Provisioner interface {
	ExecMattermostCLI(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error)
}

// Context provides the API with all necessary data and interfaces for responding to requests.
//
// It is cloned before each request, allowing per-request changes such as logger annotations.
type Context struct {
	Store       Store
	Supervisor  Supervisor
	Provisioner Provisioner
	RequestID   string
	Logger      logrus.FieldLogger
}

// Clone creates a shallow copy of context, allowing clones to apply per-request changes.
func (c *Context) Clone() *Context {
	return &Context{
		Store:       c.Store,
		Supervisor:  c.Supervisor,
		Provisioner: c.Provisioner,
		Logger:      c.Logger,
	}
}
