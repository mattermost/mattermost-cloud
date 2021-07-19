// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type stackrox struct {
	awsClient      aws.AWS
	environment    string
	provisioner    *KopsProvisioner
	kops           *kops.Cmd
	cluster        *model.Cluster
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func newStackroxHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, provisioner *KopsProvisioner, awsClient aws.AWS, kops *kops.Cmd, logger log.FieldLogger) (*stackrox, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate Stackrox handle with nil logger")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to Stackrox if the provisioner provided is nil")
	}

	if kops == nil {
		return nil, errors.New("cannot create a connection to Stackrox if the Kops command provided is nil")
	}

	return &stackrox{
		awsClient:      awsClient,
		environment:    awsClient.GetCloudEnvironmentName(),
		provisioner:    provisioner,
		kops:           kops,
		cluster:        cluster,
		logger:         logger.WithField("cluster-utility", model.StackroxCanonicalName),
		desiredVersion: desiredVersion,
	}, nil

}

func (s *stackrox) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	s.actualVersion = actualVersion
	return nil
}

func (s *stackrox) ValuesPath() string {
	if s.desiredVersion == nil {
		return ""
	}
	return s.desiredVersion.Values()
}

func (s *stackrox) CreateOrUpgrade() error {
	h := s.NewHelmDeployment()

	err := h.Update()
	if err != nil {
		return err
	}

	err = s.updateVersion(h)
	return err
}

func (s *stackrox) DesiredVersion() *model.HelmUtilityVersion {
	return s.desiredVersion
}

func (s *stackrox) ActualVersion() *model.HelmUtilityVersion {
	if s.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(s.actualVersion.Version(), "stackrox-secured-cluster-services-"),
		ValuesPath: s.actualVersion.Values(),
	}
}

func (s *stackrox) Destroy() error {
	return nil
}

func (s *stackrox) Migrate() error {
	return nil
}

func (s *stackrox) NewHelmDeployment() *helmDeployment {
	stackroxClusterName := fmt.Sprintf("cloud-%s-%s", s.environment, s.cluster.ID)

	return &helmDeployment{
		chartDeploymentName: "stackrox-secured-cluster-services",
		chartName:           "stackrox/secured-cluster-services",
		namespace:           "stackrox",
		kopsProvisioner:     s.provisioner,
		kops:                s.kops,
		logger:              s.logger,
		setArgument:         fmt.Sprintf("clusterName=%s", stackroxClusterName),
		desiredVersion:      s.desiredVersion,
	}
}

func (s *stackrox) Name() string {
	return model.StackroxCanonicalName
}
