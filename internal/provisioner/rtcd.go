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

type rtcd struct {
	awsClient      aws.AWS
	environment    string
	provisioner    *KopsProvisioner
	kops           *kops.Cmd
	cluster        *model.Cluster
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func newRtcdHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, provisioner *KopsProvisioner, awsClient aws.AWS, kops *kops.Cmd, logger log.FieldLogger) (*rtcd, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate RTCD handle with nil logger")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to RTCD if the provisioner provided is nil")
	}

	if kops == nil {
		return nil, errors.New("cannot create a connection to PgbRTCDouncer if the Kops command provided is nil")
	}

	return &rtcd{
		awsClient:      awsClient,
		environment:    awsClient.GetCloudEnvironmentName(),
		provisioner:    provisioner,
		kops:           kops,
		cluster:        cluster,
		logger:         logger.WithField("cluster-utility", model.RtcdCanonicalName),
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.Rtcd,
	}, nil

}

func (p *rtcd) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	p.actualVersion = actualVersion
	return nil
}

func (p *rtcd) ValuesPath() string {
	if p.desiredVersion == nil {
		return ""
	}
	return p.desiredVersion.Values()
}

func (p *rtcd) CreateOrUpgrade() error {
	err := p.DeployManifests()
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

func (p *rtcd) DesiredVersion() *model.HelmUtilityVersion {
	return p.desiredVersion
}

func (p *rtcd) ActualVersion() *model.HelmUtilityVersion {
	if p.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(p.actualVersion.Version(), "rtcd-"),
		ValuesPath: p.actualVersion.Values(),
	}
}

func (p *rtcd) Destroy() error {
	return nil
}

func (p *rtcd) Migrate() error {
	return nil
}

func (p *rtcd) NewHelmDeployment() *helmDeployment {
	return &helmDeployment{
		chartDeploymentName: "rtcd",
		chartName:           "chartmuseum/rtcd", #########################################################################
		namespace:           "mattermost-rtcd",
		kopsProvisioner:     p.provisioner,
		kops:                p.kops,
		logger:              p.logger,
		desiredVersion:      p.desiredVersion,
	}
}

func (p *rtcd) Name() string {
	return model.RtcdCanonicalName
}
