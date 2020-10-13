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
	desiredVersion string
	actualVersion  string
}

func newTeleportHandle(cluster *model.Cluster, desiredVersion string, provisioner *KopsProvisioner, awsClient aws.AWS, kops *kops.Cmd, logger log.FieldLogger) (*teleport, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate Teleport handle with nil logger")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to Teleport if the provisioner provided is nil")
	}

	if kops == nil {
		return nil, errors.New("cannot create a connection to Teleport if the Kops command provided is nil")
	}

	environment, err := awsClient.GetCloudEnvironmentName()
	if err != nil {
		return nil, err
	}

	if environment == "" {
		return nil, errors.New("cannot create a connection to Teleport if the environment is empty")
	}

	return &teleport{
		awsClient:      awsClient,
		environment:    environment,
		provisioner:    provisioner,
		kops:           kops,
		cluster:        cluster,
		logger:         logger.WithField("cluster-utility", model.TeleportCanonicalName),
		desiredVersion: desiredVersion,
	}, nil

}

func (t *teleport) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	t.actualVersion = actualVersion
	return nil
}

func (t *teleport) CreateOrUpgrade() error {
	h := t.NewHelmDeployment()
	err := h.Update()
	if err != nil {
		return err
	}

	err = t.updateVersion(h)
	return err
}

func (t *teleport) DesiredVersion() string {
	return t.desiredVersion
}

func (t *teleport) ActualVersion() string {
	return strings.TrimPrefix(t.actualVersion, "teleport-")
}

func (t *teleport) Destroy() error {
	teleportClusterName := fmt.Sprintf("cloud-%s-%s", t.environment, t.cluster.ID)
	err := t.awsClient.S3EnsureBucketDeleted(teleportClusterName, t.logger)
	if err != nil {
		return errors.Wrap(err, "unable to delete Teleport bucket")
	}

	err = t.awsClient.DynamoDBEnsureTableDeleted(teleportClusterName, t.logger)
	if err != nil {
		return errors.Wrap(err, "unable to delete Teleport dynamodb table")
	}

	err = t.awsClient.DynamoDBEnsureTableDeleted(fmt.Sprintf("%s-events", teleportClusterName), t.logger)
	if err != nil {
		return errors.Wrap(err, "unable to delete Teleport dynamodb events table")
	}
	return nil
}

func (t *teleport) Migrate() error {
	return nil
}

func (t *teleport) NewHelmDeployment() *helmDeployment {
	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		awsRegion = aws.DefaultAWSRegion
	}
	teleportClusterName := fmt.Sprintf("cloud-%s-%s", t.environment, t.cluster.ID)
	return &helmDeployment{
		chartDeploymentName: "teleport",
		chartName:           "chartmuseum/teleport",
		namespace:           "teleport",
		setArgument:         fmt.Sprintf("config.auth_service.cluster_name=%[1]s,config.teleport.storage.region=%[2]s,config.teleport.storage.table_name=%[1]s,config.teleport.storage.audit_events_uri=dynamodb://%[1]s-events,config.teleport.storage.audit_sessions_uri=s3://%[1]s/records?region=%[2]s", teleportClusterName, awsRegion),
		valuesPath:          t.ValuesPath(),
		kopsProvisioner:     t.provisioner,
		kops:                t.kops,
		logger:              t.logger,
		desiredVersion:      t.desiredVersion,
	}
}

func (*teleport) ValuesPath() string {
	return model.UtilityValuesDirectory() + "/teleport_values.yaml"
}

func (*teleport) Name() string {
	return model.TeleportCanonicalName
}
