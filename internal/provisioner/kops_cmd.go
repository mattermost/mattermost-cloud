package provisioner

import (
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	log "github.com/sirupsen/logrus"
)

type kopsFactoryFunc func(logger log.FieldLogger) (KopsCmd, error)

// KopsCmd describes the interface required by the provisioner to interact with kops.
type KopsCmd interface {
	CreateCluster(name string, cloud string, clusterSize kops.ClusterSize, zones []string) error
	GetCluster(name string) (string, error)
	UpdateCluster(name string) error
	UpgradeCluster(name string) error
	DeleteCluster(name string) error
	RollingUpdateCluster(name string) error
	WaitForKubernetesReadiness(dns string, timeout int) error
	ValidateCluster(name string, silent bool) error
	GetOutputDirectory() string
	GetKubeConfigPath() string
	Close() error
}
