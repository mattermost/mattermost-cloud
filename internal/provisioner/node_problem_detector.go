// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type nodeProblemDetector struct {
	provisioner    *KopsProvisioner
	awsClient      aws.AWS
	kops           *kops.Cmd
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func newNodeProblemDetectorHandle(version *model.HelmUtilityVersion, provisioner *KopsProvisioner, awsClient aws.AWS, kops *kops.Cmd, logger log.FieldLogger) (*nodeProblemDetector, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate NodeProblemDetector handle with nil logger")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to NodeProblemDetector if the provisioner provided is nil")
	}

	if awsClient == nil {
		return nil, errors.New("cannot create a connection to NodeProblemDetector if the awsClient provided is nil")
	}

	if kops == nil {
		return nil, errors.New("cannot create a connection to NodeProblemDetector if the Kops command provided is nil")
	}

	return &nodeProblemDetector{
		provisioner:    provisioner,
		awsClient:      awsClient,
		kops:           kops,
		logger:         logger.WithField("cluster-utility", model.NodeProblemDetectorCanonicalName),
		desiredVersion: version,
	}, nil
}

func (f *nodeProblemDetector) Destroy() error {
	return nil
}

func (f *nodeProblemDetector) Migrate() error {
	return nil
}

func (f *nodeProblemDetector) CreateOrUpgrade() error {
	logger := f.logger.WithField("node-problem-detector-action", "upgrade")
	h := f.NewHelmDeployment(logger)

	err := h.Update()
	if err != nil {
		return err
	}

	err = f.updateVersion(h)
	return err
}

func (f *nodeProblemDetector) DesiredVersion() *model.HelmUtilityVersion {
	return f.desiredVersion
}

func (f *nodeProblemDetector) ActualVersion() *model.HelmUtilityVersion {
	if f.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(f.actualVersion.Version(), "node-problem-detector-"),
		ValuesPath: f.actualVersion.Values(),
	}
}

func (f *nodeProblemDetector) Name() string {
	return model.NodeProblemDetectorCanonicalName
}

func (f *nodeProblemDetector) NewHelmDeployment(logger log.FieldLogger) *helmDeployment {
	return &helmDeployment{
		chartDeploymentName: "node-problem-detector",
		chartName:           "deliveryhero/node-problem-detector",
		namespace:           "node-problem-detector",
		kopsProvisioner:     f.provisioner,
		kops:                f.kops,
		logger:              f.logger,
		desiredVersion:      f.desiredVersion,
	}
}

func (f *nodeProblemDetector) ValuesPath() string {
	if f.desiredVersion == nil {
		return ""
	}
	return f.desiredVersion.Values()
}

func (f *nodeProblemDetector) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	f.actualVersion = actualVersion
	return nil
}
