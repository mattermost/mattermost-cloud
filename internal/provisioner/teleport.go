// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type teleport struct {
	awsClient      aws.AWS
	kubeconfigPath string
	environment    string
	cluster        *model.Cluster
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func newTeleportHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, kubeconfigPath string, awsClient aws.AWS, logger log.FieldLogger) (*teleport, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate Teleport handle with nil logger")
	}
	if kubeconfigPath == "" {
		return nil, errors.New("cannot create utility without kubeconfig")
	}

	return &teleport{
		awsClient:      awsClient,
		kubeconfigPath: kubeconfigPath,
		environment:    awsClient.GetCloudEnvironmentName(),
		cluster:        cluster,
		logger:         logger.WithField("cluster-utility", model.TeleportCanonicalName),
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.Teleport,
	}, nil

}

func (n *teleport) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	n.actualVersion = actualVersion
	return nil
}

func (n *teleport) ValuesPath() string {
	if n.desiredVersion == nil {
		return ""
	}
	return n.desiredVersion.Values()
}

func (n *teleport) CreateOrUpgrade() error {
	h := n.NewHelmDeployment()

	err := h.Update()
	if err != nil {
		return err
	}

	err = n.updateVersion(h)
	return err
}

func (n *teleport) DesiredVersion() *model.HelmUtilityVersion {
	return n.desiredVersion
}

func (n *teleport) ActualVersion() *model.HelmUtilityVersion {
	if n.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(n.actualVersion.Version(), "teleport-kube-agent-"),
		ValuesPath: n.actualVersion.Values(),
	}
}

func (n *teleport) Destroy() error {
	teleportClusterName := fmt.Sprintf("cloud-%s-%s", n.environment, n.cluster.ID)
	err := n.awsClient.S3EnsureBucketDeleted(teleportClusterName, n.logger)
	if err != nil {
		return errors.Wrap(err, "unable to delete Teleport bucket")
	}

	helm := n.NewHelmDeployment()
	return helm.Delete()
}

func (n *teleport) Migrate() error {
	return nil
}

func (n *teleport) NewHelmDeployment() *helmDeployment {
	teleportClusterName := fmt.Sprintf("cloud-%s-%s", n.environment, n.cluster.ID)
	return newHelmDeployment(
		"chartmuseum/teleport-kube-agent",
		"teleport-kube-agent",
		"teleport",
		n.kubeconfigPath,
		n.desiredVersion,
		fmt.Sprintf("kubeClusterName=%s", teleportClusterName),
		n.logger,
	)
}

func (n *teleport) Name() string {
	return model.TeleportCanonicalName
}
