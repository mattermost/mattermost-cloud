// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type metricsServer struct {
	provisioner    *KopsProvisioner
	kops           *kops.Cmd
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func newMetricsServerHandle(desiredVersion *model.HelmUtilityVersion, cluster *model.Cluster, provisioner *KopsProvisioner, kops *kops.Cmd, logger log.FieldLogger) (*metricsServer, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate MetricsServer handle with nil logger")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to MetricsServer if the provisioner provided is nil")
	}

	if kops == nil {
		return nil, errors.New("cannot create a connection to MetricsServer if the Kops command provided is nil")
	}

	return &metricsServer{
		provisioner:    provisioner,
		kops:           kops,
		logger:         logger.WithField("cluster-utility", model.MetricsServerCanonicalName),
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.MetricsServer,
	}, nil
}

func (m *metricsServer) Destroy() error {
	return nil
}

func (m *metricsServer) Migrate() error {
	return nil
}

func (m *metricsServer) CreateOrUpgrade() error {
	logger := m.logger.WithField("metrics-server-action", "upgrade")
	h := m.NewHelmDeployment(logger)

	err := h.Update()
	if err != nil {
		return err
	}

	err = m.updateVersion(h)
	return err
}

func (m *metricsServer) DesiredVersion() *model.HelmUtilityVersion {
	return m.desiredVersion
}

func (m *metricsServer) ActualVersion() *model.HelmUtilityVersion {
	if m.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(m.actualVersion.Version(), "metrics-server-"),
		ValuesPath: m.actualVersion.Values(),
	}
}

func (m *metricsServer) Name() string {
	return model.MetricsServerCanonicalName
}

func (m *metricsServer) NewHelmDeployment(logger log.FieldLogger) *helmDeployment {
	return &helmDeployment{
		chartDeploymentName: "metrics-server",
		chartName:           "metrics-server/metrics-server",
		namespace:           "kube-system",
		kopsProvisioner:     m.provisioner,
		kops:                m.kops,
		logger:              logger,
		desiredVersion:      m.desiredVersion,
	}
}

func (m *metricsServer) ValuesPath() string {
	if m.desiredVersion == nil {
		return ""
	}
	return m.desiredVersion.Values()
}

func (m *metricsServer) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	m.actualVersion = actualVersion
	return nil
}
