// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"github.com/mattermost/mattermost-cloud/internal/api"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

type ClusterProvisionerOption struct {
	kopsProvisioner *KopsProvisioner
	eksProvisioner  *EKSProvisioner
}

func (c ClusterProvisionerOption) GetClusterProvisioner(provisioner string) supervisor.ClusterProvisioner {
	if provisioner == model.ProvisionerEKS {
		return c.eksProvisioner
	}

	return c.kopsProvisioner
}

// ProvisioningParams represent configuration used during various provisioning operations.
type ProvisioningParams struct {
	S3StateStore                  string
	AllowCIDRRangeList            []string
	VpnCIDRList                   []string
	Owner                         string
	UseExistingAWSResources       bool
	DeployMysqlOperator           bool
	DeployMinioOperator           bool
	DeployLocalMattermostOperator bool
	NdotsValue                    string
	InternalIPRanges              []string
	PGBouncerConfig               *model.PGBouncerConfig
	SLOInstallationGroups         []string
	SLOEnterpriseGroups           []string
	EtcdManagerEnv                map[string]string
}

type Provisioner struct {
	ClusterProvisionerOption
	params         ProvisioningParams
	awsClient      aws.AWS
	resourceUtil   *utils.ResourceUtil
	backupOperator *BackupOperator
	store          *store.SQLStore
	logger         log.FieldLogger
	kubeOption     ClusterProvisionerOption
}

var _ supervisor.ClusterInstallationProvisioner = (*Provisioner)(nil)
var _ supervisor.InstallationProvisioner = (*Provisioner)(nil)
var _ supervisor.BackupProvisioner = (*Provisioner)(nil)
var _ supervisor.RestoreProvisioner = (*Provisioner)(nil)
var _ supervisor.ImportProvisioner = (*Provisioner)(nil)
var _ supervisor.DBMigrationCIProvisioner = (*Provisioner)(nil)

var _ api.Provisioner = (*Provisioner)(nil)

var _ kube = (*KopsProvisioner)(nil)

func NewProvisioner(
	kopsProvisioner *KopsProvisioner,
	eksProvisioner *EKSProvisioner,
	params ProvisioningParams,
	awsClient aws.AWS,
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
		awsClient:      awsClient,
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
