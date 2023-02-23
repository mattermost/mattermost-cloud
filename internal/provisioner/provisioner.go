package provisioner

import (
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	log "github.com/sirupsen/logrus"
)

type ClusterProvisionerOption struct {
	kopsProvisioner *KopsProvisioner
	eksProvisioner  *EKSProvisioner
}

func (c ClusterProvisionerOption) GetClusterProvisioner(provisioner string) supervisor.ClusterProvisioner {
	if provisioner == "eks" {
		return c.eksProvisioner
	}

	return c.kopsProvisioner
}

type Provisioner struct {
	ClusterProvisionerOption
	params         ProvisioningParams
	resourceUtil   *utils.ResourceUtil
	backupOperator *BackupOperator
	store          *store.SQLStore
	logger         log.FieldLogger
	kubeOption     ClusterProvisionerOption
}

func NewProvisioner(
	kopsProvisioner *KopsProvisioner,
	eksProvisioner *EKSProvisioner,
	params ProvisioningParams,
	resourceUtil *utils.ResourceUtil,
	backupOperator *BackupOperator,
	sqlStore *store.SQLStore,
	logger log.FieldLogger,
) Provisioner {

	return Provisioner{
		ClusterProvisionerOption: ClusterProvisionerOption{
			kopsProvisioner: kopsProvisioner,
			eksProvisioner:  eksProvisioner,
		},
		params:         params,
		resourceUtil:   resourceUtil,
		backupOperator: backupOperator,
		store:          sqlStore,
		logger:         logger,
		kubeOption: ClusterProvisionerOption{
			kopsProvisioner: kopsProvisioner,
			eksProvisioner:  eksProvisioner,
		},
	}
}
