// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utility

import (
	"strings"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type cloudprober struct {
	cluster        *model.Cluster
	kubeconfigPath string
	logger         log.FieldLogger
	actualVersion  *model.HelmUtilityVersion
	desiredVersion *model.HelmUtilityVersion
}

func newCloudproberOrUnmanagedHandle(cluster *model.Cluster, kubeconfigPath string, logger log.FieldLogger) (Utility, error) {
	desired := cluster.DesiredUtilityVersion(model.CloudproberCanonicalName)
	actual := cluster.ActualUtilityVersion(model.CloudproberCanonicalName)

	// if model.UtilityIsUnmanaged(desired, actual) {
	// 	return newUnmanagedHandle(model.CloudproberCanonicalName, logger), nil
	// }
	if model.UtilityIsUnmanaged(desired, actual) {
		return nil, nil
	}
	cloudprober := newCloudproberHandle(desired, cluster, kubeconfigPath, logger)
	err := cloudprober.validate()
	if err != nil {
		return nil, errors.Wrap(err, "cloudprober utility config is invalid")
	}

	return cloudprober, nil
}

func newCloudproberHandle(desiredVersion *model.HelmUtilityVersion, cluster *model.Cluster, kubeconfigPath string, logger log.FieldLogger) *cloudprober {
	return &cloudprober{
		cluster:        cluster,
		logger:         logger.WithField("cluster-utility", model.CloudproberCanonicalName),
		kubeconfigPath: kubeconfigPath,
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.Cloudprober,
	}
}

func (c *cloudprober) validate() error {
	if c.kubeconfigPath == "" {
		return errors.New("kubeconfig path cannot be empty")
	}
	if c.cluster == nil {
		return errors.New("cluster cannot be nil")
	}

	return nil
}

func (c *cloudprober) CreateOrUpgrade() error {
	logger := c.logger.WithField("cloudprober-action", "upgrade")
	h := c.newHelmDeployment(logger)

	err := h.Update()
	if err != nil {
		return err
	}

	err = c.updateVersion(h)
	return err
}

func (c *cloudprober) Name() string {
	return model.CloudproberCanonicalName
}

func (c *cloudprober) Destroy() error {
	helm := c.newHelmDeployment(c.logger)
	return helm.Delete()
}

func (c *cloudprober) Migrate() error {
	return nil
}

func (c *cloudprober) DesiredVersion() *model.HelmUtilityVersion {
	return c.desiredVersion
}

func (c *cloudprober) ActualVersion() *model.HelmUtilityVersion {
	if c.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(c.actualVersion.Version(), "cloudprober-"),
		ValuesPath: c.actualVersion.Values(),
	}
}

func (c *cloudprober) newHelmDeployment(logger log.FieldLogger) *helmDeployment {
	return newHelmDeployment(
		"chartmuseum/cloudprober",
		"cloudprober",
		"cloudprober",
		c.kubeconfigPath,
		c.desiredVersion,
		defaultHelmDeploymentSetArgument,
		logger,
	)
}

func (c *cloudprober) ValuesPath() string {
	if c.desiredVersion == nil {
		return ""
	}
	return c.desiredVersion.Values()
}

func (c *cloudprober) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	c.actualVersion = actualVersion
	return nil
}
