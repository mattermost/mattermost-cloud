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

type haproxy struct {
	environment    string
	kubeconfigPath string
	cluster        *model.Cluster
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func newHaproxyHandle(desiredVersion *model.HelmUtilityVersion, cluster *model.Cluster,kubeconfigPath string, awsClient aws.AWS, logger log.FieldLogger) (*haproxy, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate Haproxy handle with nil logger")
	}
	if awsClient == nil {
		return nil, errors.New("cannot create a connection to Haproxy if the awsClient provided is nil")
	}
	if kubeconfigPath == "" {
		return nil, errors.New("cannot create utility without kubeconfig")
	}

	return &haproxy{
		environment:    awsClient.GetCloudEnvironmentName(),
		kubeconfigPath: kubeconfigPath,
		cluster:        cluster,
		logger:         logger.WithField("cluster-utility", model.HaproxyCanonicalName),
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.Haproxy,
	}, nil

}

func (ha *haproxy) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	ha.actualVersion = actualVersion
	return nil
}

func (ha *haproxy) ValuesPath() string {
	if ha.desiredVersion == nil {
		return ""
	}
	return ha.desiredVersion.Values()
}

func (ha *haproxy) CreateOrUpgrade() error {

	h := ha.NewHelmDeployment()

	err := h.Update()
	if err != nil {
		return err
	}

	err = ha.updateVersion(h)
	return err
}

func (ha *haproxy) DesiredVersion() *model.HelmUtilityVersion {
	return ha.desiredVersion
}

func (ha *haproxy) ActualVersion() *model.HelmUtilityVersion {
	if ha.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(ha.actualVersion.Version(), "mattermost-haproxy-"),
		ValuesPath: ha.actualVersion.Values(),
	}
}

func (ha *haproxy) Destroy() error {
	// if anything needs to be deleted can be added here
	return nil
}

func (ha *haproxy) Migrate() error {
	// if anything needs to be migrated can be added here
	return nil
}

func (ha *haproxy) NewHelmDeployment() *helmDeployment {
	return &helmDeployment{
		chartDeploymentName: "kubernetes-ingress-haproxy",
		chartName:           "haproxytech/kubernetes-ingress",
		namespace:           "kubernetes-ingress-haproxy",
		kubeconfigPath:      ha.kubeconfigPath,
		logger:              ha.logger,
		desiredVersion:      ha.desiredVersion,
	}
}

func (ha *haproxy) Name() string {
	return model.HaproxyCanonicalName
}
