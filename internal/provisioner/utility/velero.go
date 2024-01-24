// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utility

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/argocd"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/git"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type velero struct {
	awsClient      aws.AWS
	gitClient      git.Client
	argocdClient   argocd.Client
	cluster        *model.Cluster
	kubeconfigPath string
	logger         log.FieldLogger
	actualVersion  *model.HelmUtilityVersion
	desiredVersion *model.HelmUtilityVersion
	tempDir        string
}

func newVeleroOrUnmanagedHandle(cluster *model.Cluster, kubeconfigPath, tempDir string, awsClient aws.AWS, gitClient git.Client, argocdClient argocd.Client, logger log.FieldLogger) (Utility, error) {
	desired := cluster.DesiredUtilityVersion(model.VeleroCanonicalName)
	actual := cluster.ActualUtilityVersion(model.VeleroCanonicalName)

	if model.UtilityIsUnmanaged(desired, actual) {
		return newUnmanagedHandle(model.VeleroCanonicalName, kubeconfigPath, tempDir, []string{}, cluster, awsClient, gitClient, argocdClient, logger), nil
	}
	velero := newVeleroHandle(desired, cluster, kubeconfigPath, tempDir, awsClient, gitClient, argocdClient, logger)
	err := velero.validate()
	if err != nil {
		return nil, errors.Wrap(err, "teleport utility config is invalid")
	}

	return velero, nil
}

func newVeleroHandle(desiredVersion *model.HelmUtilityVersion, cluster *model.Cluster, kubeconfigPath, tempDir string, awsClient aws.AWS, gitClient git.Client, argocdClient argocd.Client, logger log.FieldLogger) *velero {
	return &velero{
		awsClient:      awsClient,
		gitClient:      gitClient,
		argocdClient:   argocdClient,
		cluster:        cluster,
		kubeconfigPath: kubeconfigPath,
		logger:         logger.WithField("cluster-utility", model.VeleroCanonicalName),
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.Velero,
		tempDir:        tempDir,
	}
}

func (v *velero) validate() error {
	if v.kubeconfigPath == "" {
		return errors.New("kubeconfig path cannot be empty")
	}

	return nil
}

func (v *velero) CreateOrUpgrade() error {
	logger := v.logger.WithField("velero-action", "upgrade")
	if err := ProvisionUtilityArgocd(v.Name(), v.tempDir, v.cluster.ID, []string{}, v.awsClient, v.gitClient, v.argocdClient, logger); err != nil {
		return errors.Wrap(err, "failed to provision velero utility")
	}
	// h := v.newHelmDeployment(logger)

	// err := h.Update()
	// if err != nil {
	// 	return err
	// }

	// err = v.updateVersion(h)
	// return err
	return nil
}

func (v *velero) Name() string {
	return model.VeleroCanonicalName
}

func (v *velero) Destroy() error {
	helm := v.newHelmDeployment(v.logger)
	return helm.Delete()
}

func (v *velero) Migrate() error {
	return nil
}

func (v *velero) DesiredVersion() *model.HelmUtilityVersion {
	return v.desiredVersion
}

func (v *velero) ActualVersion() *model.HelmUtilityVersion {
	if v.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(v.actualVersion.Version(), "velero-"),
		ValuesPath: v.actualVersion.Values(),
	}
}

func (v *velero) newHelmDeployment(logger log.FieldLogger) *helmDeployment {
	helmValueArguments := fmt.Sprintf("configuration.backupStorageLocation[0].prefix=%s", v.cluster.ID)

	return newHelmDeployment(
		"vmware-tanzu/velero",
		"velero",
		"velero",
		v.kubeconfigPath,
		v.desiredVersion,
		helmValueArguments,
		logger,
	)
}

func (v *velero) ValuesPath() string {
	if v.desiredVersion == nil {
		return ""
	}
	return v.desiredVersion.Values()
}

func (v *velero) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	v.actualVersion = actualVersion
	return nil
}
