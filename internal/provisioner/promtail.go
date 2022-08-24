// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type promtail struct {
	environment    string
	kubeconfigPath string
	cluster        *model.Cluster
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func newPromtailHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, kubeconfigPath string, awsClient aws.AWS, logger log.FieldLogger) (*promtail, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate Promtail handle with nil logger")
	}
	if kubeconfigPath == "" {
		return nil, errors.New("cannot create utility without kubeconfig")
	}

	return &promtail{
		environment:    awsClient.GetCloudEnvironmentName(),
		cluster:        cluster,
		kubeconfigPath: kubeconfigPath,
		logger:         logger.WithField("cluster-utility", model.PromtailCanonicalName),
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.Promtail,
	}, nil

}

func (p *promtail) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	p.actualVersion = actualVersion
	return nil
}

func (p *promtail) ValuesPath() string {
	if p.desiredVersion == nil {
		return ""
	}
	return p.desiredVersion.Values()
}

func (p *promtail) CreateOrUpgrade() error {
	h := p.NewHelmDeployment()

	err := h.Update()
	if err != nil {
		return err
	}

	err = p.updateVersion(h)
	return err
}

func (p *promtail) DesiredVersion() *model.HelmUtilityVersion {
	return p.desiredVersion
}

func (p *promtail) ActualVersion() *model.HelmUtilityVersion {
	if p.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(p.actualVersion.Version(), "promtail-"),
		ValuesPath: p.actualVersion.Values(),
	}
}

func (p *promtail) Destroy() error {
	return nil
}

func (p *promtail) Migrate() error {
	return nil
}

func (p *promtail) NewHelmDeployment() *helmDeployment {
	return &helmDeployment{
		chartDeploymentName: "promtail",
		chartName:           "grafana/promtail",
		namespace:           "promtail",
		logger:              p.logger,
		kubeconfigPath:      p.kubeconfigPath,
		setArgument:         fmt.Sprintf("extraArgs={-client.external-labels=cluster=%s}", p.cluster.ID),
		desiredVersion:      p.desiredVersion,
	}
}

func (p *promtail) Name() string {
	return model.PromtailCanonicalName
}
