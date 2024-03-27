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

type teleport struct {
	awsClient      aws.AWS
	gitClient      git.Client
	argocdClient   argocd.Client
	kubeconfigPath string
	environment    string
	cluster        *model.Cluster
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
	tempDir        string
}

func newTeleportOrUnmanagedHandle(cluster *model.Cluster, kubeconfigPath, tempDir string, awsClient aws.AWS, gitClient git.Client, argocdClient argocd.Client, logger log.FieldLogger) (Utility, error) {
	desired := cluster.DesiredUtilityVersion(model.TeleportCanonicalName)
	actual := cluster.ActualUtilityVersion(model.TeleportCanonicalName)

	if model.UtilityIsUnmanaged(desired, actual) {
		return newUnmanagedHandle(model.TeleportCanonicalName, kubeconfigPath, tempDir, []string{}, cluster, awsClient, gitClient, argocdClient, logger), nil
	}
	teleport := newTeleportHandle(cluster, desired, kubeconfigPath, tempDir, awsClient, gitClient, argocdClient, logger)
	err := teleport.validate()
	if err != nil {
		return nil, errors.Wrap(err, "teleport utility config is invalid")
	}

	return teleport, nil
}

func newTeleportHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, kubeconfigPath, tempDir string, awsClient aws.AWS, gitClient git.Client, argocdClient argocd.Client, logger log.FieldLogger) *teleport {
	return &teleport{
		awsClient:      awsClient,
		gitClient:      gitClient,
		argocdClient:   argocdClient,
		kubeconfigPath: kubeconfigPath,
		environment:    awsClient.GetCloudEnvironmentName(),
		cluster:        cluster,
		logger:         logger.WithField("cluster-utility", model.TeleportCanonicalName),
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.Teleport,
		tempDir:        tempDir,
	}
}

func (t *teleport) validate() error {
	if t.kubeconfigPath == "" {
		return errors.New("kubeconfig path cannot be empty")
	}
	if t.awsClient == nil {
		return errors.New("awsClient cannot be nil")
	}

	return nil
}

func (t *teleport) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	t.actualVersion = actualVersion
	return nil
}

func (t *teleport) ValuesPath() string {
	if t.desiredVersion == nil {
		return ""
	}
	return t.desiredVersion.Values()
}

func (t *teleport) CreateOrUpgrade() error {
	h := t.newHelmDeployment()

	err := h.Update()
	if err != nil {
		return err
	}

	err = t.updateVersion(h)
	return err
}

func (t *teleport) DesiredVersion() *model.HelmUtilityVersion {
	return t.desiredVersion
}

func (t *teleport) ActualVersion() *model.HelmUtilityVersion {
	if t.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(t.actualVersion.Version(), "teleport-kube-agent-"),
		ValuesPath: t.actualVersion.Values(),
	}
}

func (t *teleport) Destroy() error {
	teleportClusterName := fmt.Sprintf("cloud-%s-%s", t.environment, t.cluster.ID)
	err := t.awsClient.S3EnsureBucketDeleted(teleportClusterName, t.logger)
	if err != nil {
		return errors.Wrap(err, "unable to delete Teleport bucket")
	}

	helm := t.newHelmDeployment()
	return helm.Delete()
}

func (t *teleport) Migrate() error {
	return nil
}

func (t *teleport) newHelmDeployment() *helmDeployment {
	teleportClusterName := fmt.Sprintf("cloud-%s-%s", t.environment, t.cluster.ID)
	return newHelmDeployment(
		"chartmuseum/teleport-kube-agent",
		"teleport-kube-agent",
		"teleport",
		t.kubeconfigPath,
		t.desiredVersion,
		fmt.Sprintf("kubeClusterName=%s", teleportClusterName),
		t.logger,
	)
}

func (t *teleport) Name() string {
	return model.TeleportCanonicalName
}
