// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"encoding/json"
	"strings"

	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	"github.com/mattermost/mattermost-cloud/model"
)

// KopsProvisionerType is provisioner type for Kops clusters.
const KopsProvisionerType = "kops"

// ProvisioningParams represent configuration used during various provisioning operations.
type ProvisioningParams struct {
	S3StateStore            string
	AllowCIDRRangeList      []string
	VpnCIDRList             []string
	Owner                   string
	UseExistingAWSResources bool
	DeployMysqlOperator     bool
	DeployMinioOperator     bool
	NdotsValue              string
	PGBouncerConfig         *PGBouncerConfig
	EtcdManagerEnv          map[string]string
	SLOInstallationGroups   []string
}

// KopsProvisioner provisions clusters using kops+terraform.
type KopsProvisioner struct {
	params            ProvisioningParams
	resourceUtil      *utils.ResourceUtil
	logger            log.FieldLogger
	store             model.InstallationDatabaseStoreInterface
	backupOperator    *BackupOperator
	kopsCache         map[string]*kops.Cmd
	commonProvisioner *CommonProvisioner
}

// NewKopsProvisioner creates a new KopsProvisioner.
func NewKopsProvisioner(
	provisioningParams ProvisioningParams,
	resourceUtil *utils.ResourceUtil,
	logger log.FieldLogger,
	store model.InstallationDatabaseStoreInterface,
	backupOperator *BackupOperator) *KopsProvisioner {
	logger = logger.WithField("provisioner", "kops")

	return &KopsProvisioner{
		params:         provisioningParams,
		logger:         logger,
		resourceUtil:   resourceUtil,
		store:          store,
		backupOperator: backupOperator,
		kopsCache:      make(map[string]*kops.Cmd),
		commonProvisioner: &CommonProvisioner{
			resourceUtil: resourceUtil,
			store:        store,
			params:       provisioningParams,
		},
	}
}

// ProvisionerType returns type of the provisioner.
func (provisioner *KopsProvisioner) ProvisionerType() string {
	return KopsProvisionerType
}

// Teardown cleans up cached kops provisioner data.
func (provisioner *KopsProvisioner) Teardown() {
	provisioner.logger.Debug("Performing kops provisioner cleanup")
	for name, kops := range provisioner.kopsCache {
		provisioner.logger.Debugf("Cleaning up kops cache for %s", name)
		kops.Close()
	}
}

// getKopsClusterConfigLocationFromCache returns the cached kubecfg for a k8s
// cluster. If the config is not cached, it is fetched with kops.
func (provisioner *KopsProvisioner) getCachedKopsClusterKubecfg(name string, logger log.FieldLogger) (string, error) {
	kopsClient, err := provisioner.getCachedKopsClient(name, logger)
	if err != nil {
		return "", errors.Wrap(err, "failed to get cached kops client")
	}

	return kopsClient.GetKubeConfigPath(), nil
}

func (provisioner *KopsProvisioner) getCachedKopsClient(name string, logger log.FieldLogger) (*kops.Cmd, error) {
	if kopsClient, ok := provisioner.kopsCache[name]; ok {
		logger.Debugf("Using cached kops client for %s", name)
		kopsClient.SetLogger(logger)
		return kopsClient, nil
	}

	logger.Debugf("Building kops client cache for %s", name)
	kopsClient, err := kops.New(provisioner.params.S3StateStore, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create kops wrapper")
	}
	err = kopsClient.ExportKubecfg(name)
	if err != nil {
		return nil, errors.Wrap(err, "failed to export kubecfg")
	}

	provisioner.kopsCache[name] = kopsClient
	logger.Debugf("Kops config cached at %s for %s", kopsClient.GetKubeConfigPath(), name)

	return kopsClient, nil
}

func (provisioner *KopsProvisioner) kopsClusterExists(name string, logger log.FieldLogger) (bool, error) {
	kopsClient, err := kops.New(provisioner.params.S3StateStore, logger)
	if err != nil {
		return false, errors.Wrap(err, "failed to create kops client wrapper")
	}

	clustersJSON, err := kopsClient.GetClustersJSON()
	if err != nil {
		return false, errors.Wrap(err, "failed to list clusters with kops")
	}

	kopsClusters, err := unmarshalKopsListClustersResponse(clustersJSON)
	if err != nil {
		return false, err
	}

	for _, cluster := range kopsClusters {
		if cluster.Metadata.Name == name {
			return true, nil
		}
	}

	return false, nil
}

type kopsCluster struct {
	Metadata model.KopsMetadata
}

// unmarshalKopsListClustersResponse unmarshals response from `kops get clusters -o json`.
// Kops output from this command is not consistent, and it behaves in the following ways:
//   - If there are multiple clusters an array of clusters is returned.
//   - If there is only one cluster a single cluster object is returned (not as an array).
func unmarshalKopsListClustersResponse(output string) ([]kopsCluster, error) {
	trimmedOut := strings.TrimSpace(output)
	if strings.HasPrefix(trimmedOut, "[") {
		var kopsClusters []kopsCluster
		err := json.Unmarshal([]byte(output), &kopsClusters)
		if err != nil {
			return nil, errors.Wrap(err, "failed to unmarshal array of kops clusters output")
		}
		return kopsClusters, nil
	}

	singleCluster := kopsCluster{}
	err := json.Unmarshal([]byte(output), &singleCluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal single kops cluster output")
	}
	return []kopsCluster{singleCluster}, nil
}

func (provisioner *KopsProvisioner) invalidateCachedKopsClient(name string, logger log.FieldLogger) error {
	kopsClient, ok := provisioner.kopsCache[name]
	if !ok {
		logger.Errorf("Could not find kops client cache for %s to invalidate", name)
		return errors.Errorf("could not find kops client cache for %s to invalidate", name)
	}

	logger.Debugf("Invalidating kops client cache for %s and cleaning up %s", name, kopsClient.GetOutputDirectory())
	kopsClient.Close()
	delete(provisioner.kopsCache, name)

	return nil
}

// invalidateCachedKopsClientOnError can be used to invalidate cache when the
// provided error is not nil. This can be used with defer to perform cache
// cleanup if an error is encountered that may have been due to a bad cached config.
func (provisioner *KopsProvisioner) invalidateCachedKopsClientOnError(err error, name string, logger log.FieldLogger) {
	if err == nil {
		return
	}

	provisioner.invalidateCachedKopsClient(name, logger)
}

func (provisioner *KopsProvisioner) k8sClient(clusterName string, logger log.FieldLogger) (*k8s.KubeClient, func(err error), error) {
	configLocation, err := provisioner.getCachedKopsClusterKubecfg(clusterName, logger)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to get kops config from cache")
	}
	invalidateOnError := func(err error) {
		provisioner.invalidateCachedKopsClientOnError(err, clusterName, logger)
	}
	defer invalidateOnError(err)

	var k8sClient *k8s.KubeClient
	k8sClient, err = k8s.NewFromFile(configLocation, logger)
	if err != nil {
		return nil, nil, errors.Wrap(err, "failed to create k8s client from file")
	}

	return k8sClient, invalidateOnError, nil
}
