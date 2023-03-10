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

type metricsServer struct {
	kubeconfigPath string
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
	provisioner    string
}

func newMetricsServerHandle(desiredVersion *model.HelmUtilityVersion, cluster *model.Cluster, kubeconfigPath string, logger log.FieldLogger) (*metricsServer, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate MetricsServer handle with nil logger")
	}
	if kubeconfigPath == "" {
		return nil, errors.New("cannot create utility without kubeconfig")
	}

	return &metricsServer{
		kubeconfigPath: kubeconfigPath,
		logger:         logger.WithField("cluster-utility", model.MetricsServerCanonicalName),
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.MetricsServer,
		provisioner:    cluster.Provisioner,
	}, nil
}

func (m *metricsServer) Destroy() error {
	helm := m.newHelmDeployment(m.logger)
	return helm.Delete()
}

func (m *metricsServer) Migrate() error {
	return nil
}

func (m *metricsServer) CreateOrUpgrade() error {
	logger := m.logger.WithField("metrics-server-action", "upgrade")
	h := m.newHelmDeployment(logger)

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

func (m *metricsServer) newHelmDeployment(logger log.FieldLogger) *helmDeployment {

	var setArguments []string

	if m.provisioner == model.ProvisionerEKS {
		// Calico networking cannot currently be installed on the EKS control plane nodes.
		// As a result the control plane nodes will not be able to initiate network connections to Calico pods.
		// As a workaround, trusted pods that require control plane nodes to connect to them,
		// such as those implementing admission controller webhooks, can include hostNetwork:true in their pod spec.
		// See https://docs.tigera.io/calico/3.25/getting-started/kubernetes/managed-public-cloud/eks

		setArguments = append(setArguments, "hostNetwork.enabled=true")
	}

	return newHelmDeployment(
		"metrics-server/metrics-server",
		"metrics-server",
		"kube-system",
		m.kubeconfigPath,
		m.desiredVersion,
		strings.Join(setArguments, ","),
		logger,
	)
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
