// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package helm

import (
	"fmt"
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

func NewCloudproberHandle(desiredVersion *model.HelmUtilityVersion, cluster *model.Cluster, kubeconfigPath string, logger log.FieldLogger) (*cloudprober, error) {
	if logger == nil {
		return nil, fmt.Errorf("cannot instantiate Cloudprober handle with nil logger")
	}

	if cluster == nil {
		return nil, errors.New("cannot create a connection to Cloudprober if the cluster provided is nil")
	}
	if kubeconfigPath == "" {
		return nil, errors.New("cannot create utility without kubeconfig")
	}

	return &cloudprober{
		cluster:        cluster,
		logger:         logger.WithField("cluster-utility", model.CloudproberCanonicalName),
		kubeconfigPath: kubeconfigPath,
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.Cloudprober,
	}, nil
}

func (f *cloudprober) CreateOrUpgrade() error {
	logger := f.logger.WithField("cloudprober-action", "upgrade")
	h := f.NewHelmDeployment(logger)

	err := h.Update()
	if err != nil {
		return err
	}

	err = f.updateVersion(h)
	return err
}

func (f *cloudprober) Name() string {
	return model.CloudproberCanonicalName
}

func (f *cloudprober) Destroy() error {
	helm := f.NewHelmDeployment(f.logger)
	return helm.Delete()
}

func (f *cloudprober) Migrate() error {
	return nil
}

func (f *cloudprober) DesiredVersion() *model.HelmUtilityVersion {
	return f.desiredVersion
}

func (f *cloudprober) ActualVersion() *model.HelmUtilityVersion {
	if f.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(f.actualVersion.Version(), "cloudprober-"),
		ValuesPath: f.actualVersion.Values(),
	}
}

func (f *cloudprober) NewHelmDeployment(logger log.FieldLogger) *helmDeployment {
	return newHelmDeployment(
		"chartmuseum/cloudprober",
		"cloudprober",
		"cloudprober",
		f.kubeconfigPath,
		f.desiredVersion,
		defaultHelmDeploymentSetArgument,
		logger,
	)
}

func (f *cloudprober) ValuesPath() string {
	if f.desiredVersion == nil {
		return ""
	}
	return f.desiredVersion.Values()
}

func (f *cloudprober) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	f.actualVersion = actualVersion
	return nil
}
