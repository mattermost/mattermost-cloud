// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	"github.com/mattermost/mattermost-cloud/model"
)

// KopsProvisioner provisions clusters using kops+terraform.
type KopsProvisioner struct {
	s3StateStore            string
	allowCIDRRangeList      []string
	vpnCIDRList             []string
	owner                   string
	useExistingAWSResources bool
	resourceUtil            *utils.ResourceUtil
	logger                  log.FieldLogger
	store                   model.InstallationDatabaseStoreInterface
	kopsCache               map[string]*kops.Cmd
}

// NewKopsProvisioner creates a new KopsProvisioner.
func NewKopsProvisioner(s3StateStore, owner string, useExistingAWSResources bool, allowCIDRRangeList, vpnCIDRList []string,
	resourceUtil *utils.ResourceUtil, logger log.FieldLogger, store model.InstallationDatabaseStoreInterface) *KopsProvisioner {
	logger = logger.WithField("provisioner", "kops")

	return &KopsProvisioner{
		s3StateStore:            s3StateStore,
		useExistingAWSResources: useExistingAWSResources,
		allowCIDRRangeList:      allowCIDRRangeList,
		vpnCIDRList:             vpnCIDRList,
		logger:                  logger,
		resourceUtil:            resourceUtil,
		owner:                   owner,
		store:                   store,
		kopsCache:               make(map[string]*kops.Cmd),
	}
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
	kopsClient, err := kops.New(provisioner.s3StateStore, logger)
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
