// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utility

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

func newPromtailOrUnmanagedHandle(cluster *model.Cluster, kubeconfigPath string, awsClient aws.AWS, logger log.FieldLogger) (Utility, error) {
	desired := cluster.DesiredUtilityVersion(model.PromtailCanonicalName)
	actual := cluster.ActualUtilityVersion(model.PromtailCanonicalName)

	if model.UtilityIsUnmanaged(desired, actual) {
		return newUnmanagedHandle(model.PromtailCanonicalName, logger), nil
	}
	promtail := newPromtailHandle(cluster, desired, kubeconfigPath, awsClient, logger)
	err := promtail.validate()
	if err != nil {
		return nil, errors.Wrap(err, "promtail utility config is invalid")
	}

	return promtail, nil
}

func newPromtailHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, kubeconfigPath string, awsClient aws.AWS, logger log.FieldLogger) *promtail {
	return &promtail{
		environment:    awsClient.GetCloudEnvironmentName(),
		cluster:        cluster,
		kubeconfigPath: kubeconfigPath,
		logger:         logger.WithField("cluster-utility", model.PromtailCanonicalName),
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.Promtail,
	}
}

func (p *promtail) validate() error {
	if p.kubeconfigPath == "" {
		return errors.New("kubeconfig path cannot be empty")
	}

	return nil
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
	h := p.newHelmDeployment()

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
	helm := p.newHelmDeployment()
	return helm.Delete()
}

func (p *promtail) Migrate() error {
	return nil
}

func (p *promtail) newHelmDeployment() *helmDeployment {
	return newHelmDeployment(
		"grafana/promtail",
		"promtail",
		"promtail",
		p.kubeconfigPath,
		p.desiredVersion,
		fmt.Sprintf("extraArgs={-client.external-labels=cluster=%s}", p.cluster.ID),
		p.logger,
	)
}

func (p *promtail) Name() string {
	return model.PromtailCanonicalName
}
