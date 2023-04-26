// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utility

import (
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	chartName      = "bitpoke/mysql-operator"
	deploymentName = "mysql-operator"
)

type mysqlOperator struct {
	environment    string
	kubeconfigPath string
	cluster        *model.Cluster
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func newMysqlOperatorHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, kubeconfigPath string, awsClient aws.AWS, logger log.FieldLogger) (*mysqlOperator, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate mysql operator handle with nil logger")
	}
	if awsClient == nil {
		return nil, errors.New("cannot create a connection to mysql opeator if the awsClient provided is nil")
	}
	if kubeconfigPath == "" {
		return nil, errors.New("cannot create utility without kubeconfig")
	}

	return &mysqlOperator{
		environment:    awsClient.GetCloudEnvironmentName(),
		kubeconfigPath: kubeconfigPath,
		cluster:        cluster,
		logger:         logger.WithField("cluster-utility", model.MysqlOperatorCanonicalName),
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.MysqlOperator,
	}, nil

}

func (r *mysqlOperator) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	r.actualVersion = actualVersion
	return nil
}

func (r *mysqlOperator) ValuesPath() string {
	if r.desiredVersion == nil {
		return ""
	}
	return r.desiredVersion.Values()
}

func (r *mysqlOperator) CreateOrUpgrade() error {

	h := r.newHelmDeployment()

	err := h.Update()
	if err != nil {
		return err
	}

	err = r.updateVersion(h)
	return err
}

func (r *mysqlOperator) DesiredVersion() *model.HelmUtilityVersion {
	return r.desiredVersion
}

func (r *mysqlOperator) ActualVersion() *model.HelmUtilityVersion {
	if r.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(r.actualVersion.Version(), "mysql-operator"),
		ValuesPath: r.actualVersion.Values(),
	}
}

func (r *mysqlOperator) Destroy() error {
	helm := r.newHelmDeployment()
	return helm.Delete()
}

func (r *mysqlOperator) Migrate() error {
	// if anything needs to be migrated can be added here
	return nil
}

func (r *mysqlOperator) newHelmDeployment() *helmDeployment {
	return newHelmDeployment(
		chartName,
		deploymentName,
		deploymentName,
		r.kubeconfigPath,
		r.desiredVersion,
		defaultHelmDeploymentSetArgument,
		r.logger,
	)
}

func (r *mysqlOperator) Name() string {
	return model.MysqlOperatorCanonicalName
}
