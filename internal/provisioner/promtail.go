// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type promtail struct {
	environment    string
	provisioner    *KopsProvisioner
	kops           *kops.Cmd
	cluster        *model.Cluster
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func newPromtailHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, provisioner *KopsProvisioner, awsClient aws.AWS, kops *kops.Cmd, logger log.FieldLogger) (*promtail, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate Promtail handle with nil logger")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to Promtail if the provisioner provided is nil")
	}

	if kops == nil {
		return nil, errors.New("cannot create a connection to Promtail if the Kops command provided is nil")
	}

	return &promtail{
		environment:    awsClient.GetCloudEnvironmentName(),
		provisioner:    provisioner,
		kops:           kops,
		cluster:        cluster,
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
		kopsProvisioner:     p.provisioner,
		kops:                p.kops,
		logger:              p.logger,
		setArgument:         fmt.Sprintf("extraArgs={-client.external-labels=cluster=%s}", p.cluster.ID),
		desiredVersion:      p.desiredVersion,
	}
}

func (p *promtail) Name() string {
	return model.PromtailCanonicalName
}
