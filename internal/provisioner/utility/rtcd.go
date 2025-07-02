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

type rtcd struct {
	environment    string
	kubeconfigPath string
	cluster        *model.Cluster
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func newRtcdOrUnmanagedHandle(cluster *model.Cluster, kubeconfigPath string, awsClient aws.AWS, logger log.FieldLogger) (Utility, error) {
	desired := cluster.DesiredUtilityVersion(model.RtcdCanonicalName)
	actual := cluster.ActualUtilityVersion(model.RtcdCanonicalName)

	if model.UtilityIsUnmanaged(desired, actual) {
		return newUnmanagedHandle(model.RtcdCanonicalName, kubeconfigPath, []string{}, cluster, awsClient, logger), nil
	}

	rtcd := newRtcdHandle(cluster, desired, kubeconfigPath, awsClient, logger)
	err := rtcd.validate()
	if err != nil {
		return nil, errors.Wrap(err, "rtcd utility config is invalid")
	}

	return rtcd, nil
}

func newRtcdHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, kubeconfigPath string, awsClient aws.AWS, logger log.FieldLogger) *rtcd {
	return &rtcd{
		environment:    awsClient.GetCloudEnvironmentName(),
		kubeconfigPath: kubeconfigPath,
		cluster:        cluster,
		logger:         logger.WithField("cluster-utility", model.RtcdCanonicalName),
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.Rtcd,
	}
}

func (r *rtcd) validate() error {
	if r.kubeconfigPath == "" {
		return errors.New("kubeconfig path cannot be empty")
	}

	return nil
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

	h := r.newHelmDeployment()

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
	helm := r.newHelmDeployment()
	return helm.Delete()
}

func (r *rtcd) Migrate() error {
	// if anything needs to be migrated can be added here
	return nil
}

func (r *rtcd) newHelmDeployment() *helmDeployment {
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
