// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package helm

import (
	"strings"

	pg "github.com/mattermost/mattermost-cloud/internal/provisioner/pgbouncer"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type pgbouncer struct {
	awsClient      aws.AWS
	environment    string
	kubeconfigPath string
	cluster        *model.Cluster
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func NewPgbouncerHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, kubeconfigPath string, awsClient aws.AWS, logger log.FieldLogger) (*pgbouncer, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate Pgbouncer handle with nil logger")
	}
	if kubeconfigPath == "" {
		return nil, errors.New("cannot create helm without kubeconfig")
	}

	return &pgbouncer{
		awsClient:      awsClient,
		environment:    awsClient.GetCloudEnvironmentName(),
		cluster:        cluster,
		kubeconfigPath: kubeconfigPath,
		logger:         logger.WithField("cluster-helm", model.PgbouncerCanonicalName),
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.Pgbouncer,
	}, nil

}

func (p *pgbouncer) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	p.actualVersion = actualVersion
	return nil
}

func (p *pgbouncer) ValuesPath() string {
	if p.desiredVersion == nil {
		return ""
	}
	return p.desiredVersion.Values()
}

func (p *pgbouncer) CreateOrUpgrade() error {
	k8sClient, err := k8s.NewFromFile(p.kubeconfigPath, p.logger)
	if err != nil {
		return errors.Wrap(err, "failed to set up the k8s client")
	}

	err = pg.DeployManifests(k8sClient, p.logger)
	if err != nil {
		return err
	}

	h := p.NewHelmDeployment()

	err = h.Update()
	if err != nil {
		return err
	}

	err = p.updateVersion(h)
	return err
}

func (p *pgbouncer) DesiredVersion() *model.HelmUtilityVersion {
	return p.desiredVersion
}

func (p *pgbouncer) ActualVersion() *model.HelmUtilityVersion {
	if p.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(p.actualVersion.Version(), "pgbouncer-"),
		ValuesPath: p.actualVersion.Values(),
	}
}

func (p *pgbouncer) Destroy() error {
	helm := p.NewHelmDeployment()
	return helm.Delete()
}

func (p *pgbouncer) Migrate() error {
	return nil
}

func (p *pgbouncer) Name() string {
	return model.PgbouncerCanonicalName
}

func (p *pgbouncer) NewHelmDeployment() *helmDeployment {
	return newHelmDeployment(
		"chartmuseum/pgbouncer",
		"pgbouncer",
		"pgbouncer",
		p.kubeconfigPath,
		p.desiredVersion,
		defaultHelmDeploymentSetArgument,
		p.logger,
	)
}
