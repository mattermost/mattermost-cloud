// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type pgbouncer struct {
	awsClient      aws.AWS
	environment    string
	provisioner    *KopsProvisioner
	kops           *kops.Cmd
	cluster        *model.Cluster
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func newPgbouncerHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, provisioner *KopsProvisioner, awsClient aws.AWS, kops *kops.Cmd, logger log.FieldLogger) (*pgbouncer, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate Pgbouncer handle with nil logger")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to Pgbouncer if the provisioner provided is nil")
	}

	if kops == nil {
		return nil, errors.New("cannot create a connection to Pgbouncer if the Kops command provided is nil")
	}

	return &pgbouncer{
		awsClient:      awsClient,
		environment:    awsClient.GetCloudEnvironmentName(),
		provisioner:    provisioner,
		kops:           kops,
		cluster:        cluster,
		logger:         logger.WithField("cluster-utility", model.PgbouncerCanonicalName),
		desiredVersion: desiredVersion,
	}, nil

}

func (n *pgbouncer) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	n.actualVersion = actualVersion
	return nil
}

func (n *pgbouncer) ValuesPath() string {
	if n.desiredVersion == nil {
		return ""
	}
	return n.desiredVersion.Values()
}

func (n *pgbouncer) CreateOrUpgrade() error {
	h := n.NewHelmDeployment()

	err := h.Update()
	if err != nil {
		return err
	}

	err = n.updateVersion(h)
	return err
}

func (n *pgbouncer) DesiredVersion() *model.HelmUtilityVersion {
	return n.desiredVersion
}

func (n *pgbouncer) ActualVersion() *model.HelmUtilityVersion {
	if n.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(n.actualVersion.Version(), "pgbouncer-"),
		ValuesPath: n.actualVersion.Values(),
	}
}

func (n *pgbouncer) Destroy() error {
	return nil
}

func (n *pgbouncer) Migrate() error {
	return nil
}

func (n *pgbouncer) NewHelmDeployment() *helmDeployment {
	return &helmDeployment{
		chartDeploymentName: "pgbouncer",
		chartName:           "chartmuseum/pgbouncer",
		namespace:           "pgbouncer",
		kopsProvisioner:     n.provisioner,
		kops:                n.kops,
		logger:              n.logger,
		desiredVersion:      n.desiredVersion,
	}
}

func (n *pgbouncer) Name() string {
	return model.PgbouncerCanonicalName
}
