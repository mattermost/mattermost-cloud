package supervisor

import (
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

// ClusterProvisioner abstracts the provisioning operations required by the cluster supervisor.
type ClusterProvisioner interface {
	PrepareCluster(cluster *model.Cluster) bool
	CreateCluster(cluster *model.Cluster, aws aws.AWS) error
	CheckClusterCreated(cluster *model.Cluster, awsClient aws.AWS) (bool, error)
	CheckNodesCreated(cluster *model.Cluster, awsClient aws.AWS) (bool, error)
	ProvisionCluster(cluster *model.Cluster, aws aws.AWS) error
	UpgradeCluster(cluster *model.Cluster, aws aws.AWS) error
	ResizeCluster(cluster *model.Cluster, aws aws.AWS) error
	DeleteCluster(cluster *model.Cluster, aws aws.AWS) (bool, error)
	RefreshKopsMetadata(cluster *model.Cluster) error
	GetKubeConfigPath(cluster *model.Cluster) (string, error)
	GetKubeClient(cluster *model.Cluster) (*k8s.KubeClient, error)
}

type ClusterProvisionerOption interface {
	GetClusterProvisioner(provisioner string) ClusterProvisioner
}

// ClusterInstallationProvisioner is an interface for provisioning and managing ClusterInstallations.
type ClusterInstallationProvisioner interface {
	CreateClusterInstallation(cluster *model.Cluster, installation *model.Installation, installationDNS []*model.InstallationDNS, clusterInstallation *model.ClusterInstallation) error
	EnsureCRMigrated(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation) (bool, error)
	HibernateClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error
	UpdateClusterInstallation(cluster *model.Cluster, installation *model.Installation, installationDNS []*model.InstallationDNS, clusterInstallation *model.ClusterInstallation) error
	DeleteOldClusterInstallationLicenseSecrets(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error
	DeleteClusterInstallation(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error
	IsResourceReadyAndStable(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation) (bool, bool, error)
	RefreshSecrets(cluster *model.Cluster, installation *model.Installation, clusterInstallation *model.ClusterInstallation) error
	PrepareClusterUtilities(cluster *model.Cluster, installation *model.Installation, store model.ClusterUtilityDatabaseStoreInterface, awsClient aws.AWS) error
}

// InstallationProvisioner abstracts the provisioning operations required by the installation supervisor.
type InstallationProvisioner interface {
	ClusterInstallationProvisioner(version string) ClusterInstallationProvisioner
	GetClusterResources(cluster *model.Cluster, canSchedule bool, logger log.FieldLogger) (*k8s.ClusterResources, error)
	GetPublicLoadBalancerEndpoint(cluster *model.Cluster, namespace string) (string, error)
}

// BackupProvisioner provisions backup jobs on a cluster.
type BackupProvisioner interface {
	TriggerBackup(backupMeta *model.InstallationBackup, cluster *model.Cluster, installation *model.Installation) (*model.S3DataResidence, error)
	CheckBackupStatus(backupMeta *model.InstallationBackup, cluster *model.Cluster) (int64, error)
	CleanupBackupJob(backup *model.InstallationBackup, cluster *model.Cluster) error
}

// RestoreProvisioner abstracts different restoration operations required by the installation db restoration supervisor.
type RestoreProvisioner interface {
	TriggerRestore(installation *model.Installation, backup *model.InstallationBackup, cluster *model.Cluster) error
	CheckRestoreStatus(backupMeta *model.InstallationBackup, cluster *model.Cluster) (int64, error)
	CleanupRestoreJob(backup *model.InstallationBackup, cluster *model.Cluster) error
}

type ImportProvisioner interface {
	ExecMMCTL(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error)
}

type DBMigrationCIProvisioner interface {
	ClusterInstallationProvisioner(version string) ClusterInstallationProvisioner
	ExecClusterInstallationJob(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) error
}

type Provisioner interface {
	ClusterProvisionerOption
	InstallationProvisioner
	ClusterInstallationProvisioner
	BackupProvisioner
	RestoreProvisioner
	ImportProvisioner
	DBMigrationCIProvisioner
}
