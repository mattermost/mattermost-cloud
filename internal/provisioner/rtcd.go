// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type rtcd struct {
	environment    string
	kubeconfigPath string
	cluster        *model.Cluster
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func newRtcdHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, kubeconfigPath string, awsClient aws.AWS, logger log.FieldLogger) (*rtcd, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate RTCD handle with nil logger")
	}
	if awsClient == nil {
		return nil, errors.New("cannot create a connection to RTCD if the awsClient provided is nil")
	}
	if kubeconfigPath == "" {
		return nil, errors.New("cannot create utility without kubeconfig")
	}

	return &rtcd{
		environment:    awsClient.GetCloudEnvironmentName(),
		kubeconfigPath: kubeconfigPath,
		cluster:        cluster,
		logger:         logger.WithField("cluster-utility", model.RtcdCanonicalName),
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.Rtcd,
	}, nil

}

func (r *rtcd) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	r.actualVersion = actualVersion
	return nil
}

func (r *rtcd) ValuesPath() string {
	if r.desiredVersion == nil {
		return ""
	}
	return r.desiredVersion.Values()
}

func (r *rtcd) CreateOrUpgrade() error {

	h := r.NewHelmDeployment()

	err := h.Update()
	if err != nil {
		return err
	}

	err = r.updateVersion(h)
	return err
}

func (r *rtcd) DesiredVersion() *model.HelmUtilityVersion {
	return r.desiredVersion
}

func (r *rtcd) ActualVersion() *model.HelmUtilityVersion {
	if r.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(r.actualVersion.Version(), "mattermost-rtcd-"),
		ValuesPath: r.actualVersion.Values(),
	}
}

func (r *rtcd) Destroy() error {
	// if anything needs to be deleted can be added here
	return nil
}

func (r *rtcd) Migrate() error {
	// if anything needs to be migrated can be added here
	return nil
}

func (r *rtcd) NewHelmDeployment() *helmDeployment {
	return newHelmDeployment(
		"mattermost/mattermost-rtcd",
		"mattermost-rtcd",
		"mattermost-rtcd",
		r.kubeconfigPath,
		r.desiredVersion,
		defaultHelmDeploymentSetArgument,
		r.logger,
	)
}

func (r *rtcd) Name() string {
	return model.RtcdCanonicalName
}
