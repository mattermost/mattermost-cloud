// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"fmt"
	"os"
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type teleport struct {
	awsClient      aws.AWS
	environment    string
	provisioner    *KopsProvisioner
	kops           *kops.Cmd
	cluster        *model.Cluster
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func newTeleportHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, provisioner *KopsProvisioner, awsClient aws.AWS, kops *kops.Cmd, logger log.FieldLogger) (*teleport, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate Teleport handle with nil logger")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to Teleport if the provisioner provided is nil")
	}

	if kops == nil {
		return nil, errors.New("cannot create a connection to Teleport if the Kops command provided is nil")
	}

	return &teleport{
		awsClient:      awsClient,
		environment:    awsClient.GetCloudEnvironmentName(),
		provisioner:    provisioner,
		kops:           kops,
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
		Chart:      strings.TrimPrefix(n.actualVersion.Version(), "teleport-"),
		ValuesPath: n.actualVersion.Values(),
	}
}

func (n *teleport) Destroy() error {
	teleportClusterName := fmt.Sprintf("cloud-%s-%s", n.environment, n.cluster.ID)
	err := n.awsClient.S3EnsureBucketDeleted(teleportClusterName, n.logger)
	if err != nil {
		return errors.Wrap(err, "unable to delete Teleport bucket")
	}

	err = n.awsClient.DynamoDBEnsureTableDeleted(teleportClusterName, n.logger)
	if err != nil {
		return errors.Wrap(err, "unable to delete Teleport dynamodb table")
	}

	err = n.awsClient.DynamoDBEnsureTableDeleted(fmt.Sprintf("%s-events", teleportClusterName), n.logger)
	if err != nil {
		return errors.Wrap(err, "unable to delete Teleport dynamodb events table")
	}
	return nil
}

func (n *teleport) Migrate() error {
	return nil
}

func (n *teleport) NewHelmDeployment() *helmDeployment {
	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		awsRegion = aws.DefaultAWSRegion
	}
	teleportClusterName := fmt.Sprintf("cloud-%s-%s", n.environment, n.cluster.ID)
	return &helmDeployment{
		chartDeploymentName: "teleport-kube-agent",
		chartName:           "chartmuseum/teleport-kube-agent",
		namespace:           "teleport",
		setArgument:         fmt.Sprintf("kubeClusterName=%s", teleportClusterName),
		kopsProvisioner:     n.provisioner,
		kops:                n.kops,
		logger:              n.logger,
		desiredVersion:      n.desiredVersion,
	}
}

func (n *teleport) Name() string {
	return model.TeleportCanonicalName
}
