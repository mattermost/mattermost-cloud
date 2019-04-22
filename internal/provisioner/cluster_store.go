package provisioner

import "github.com/mattermost/mattermost-cloud/internal/model"

// clusterStore abstracts the database operations required to manage clusters.
type clusterStore interface {
	GetCluster(id string) (*model.Cluster, error)
	CreateCluster(cluster *model.Cluster) error
	UpdateCluster(cluster *model.Cluster) error
	DeleteCluster(id string) error
}
