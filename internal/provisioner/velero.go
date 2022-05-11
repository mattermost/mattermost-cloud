// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type velero struct {
	cluster        *model.Cluster
	kops           *kops.Cmd
	logger         log.FieldLogger
	provisioner    *KopsProvisioner
	actualVersion  *model.HelmUtilityVersion
	desiredVersion *model.HelmUtilityVersion
}

func newVeleroHandle(desiredVersion *model.HelmUtilityVersion, cluster *model.Cluster, provisioner *KopsProvisioner, kops *kops.Cmd, logger log.FieldLogger) (*velero, error) {
	if logger == nil {
		return nil, fmt.Errorf("cannot instantiate Velero handle with nil logger")
	}

	if cluster == nil {
		return nil, errors.New("cannot create a connection to Velero if the cluster provided is nil")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to Velero if the provisioner provided is nil")
	}
	if kops == nil {
		return nil, errors.New("cannot create a connection to Velero if the Kops command provided is nil")
	}
	return &velero{
		cluster:        cluster,
		kops:           kops,
		logger:         logger.WithField("cluster-utility", model.VeleroCanonicalName),
		provisioner:    provisioner,
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.Velero,
	}, nil
}

func (f *velero) Destroy() error {
	return nil
}

func (f *velero) Migrate() error {
	return nil
}

func (f *velero) CreateOrUpgrade() error {
	logger := f.logger.WithField("velero-action", "upgrade")
	h := f.NewHelmDeployment(logger)

	err := h.Update()
	if err != nil {
		return err
	}

	err = f.updateVersion(h)
	return err
}

func (f *velero) DesiredVersion() *model.HelmUtilityVersion {
	return f.desiredVersion
}

func (f *velero) ActualVersion() *model.HelmUtilityVersion {
	if f.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(f.actualVersion.Version(), "velero-"),
		ValuesPath: f.actualVersion.Values(),
	}
}

func (f *velero) Name() string {
	return model.VeleroCanonicalName
}

func (f *velero) NewHelmDeployment(logger log.FieldLogger) *helmDeployment {
	return &helmDeployment{
		chartDeploymentName: "velero",
		chartName:           "vmware-tanzu/velero",
		namespace:           "velero",
		kopsProvisioner:     f.provisioner,
		kops:                f.kops,
		logger:              logger,
		desiredVersion:      f.desiredVersion,
	}
}

func (f *velero) ValuesPath() string {
	if f.desiredVersion == nil {
		return ""
	}
	return f.desiredVersion.Values()
}

func (f *velero) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	f.actualVersion = actualVersion
	return nil
}
