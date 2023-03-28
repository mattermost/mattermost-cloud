// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utility

import (
	"strings"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type nodeProblemDetector struct {
	kubeconfigPath string
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func newNodeProblemDetectorHandle(desiredVersion *model.HelmUtilityVersion, cluster *model.Cluster, kubeconfigPath string, logger log.FieldLogger) (*nodeProblemDetector, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate NodeProblemDetector handle with nil logger")
	}
	if kubeconfigPath == "" {
		return nil, errors.New("cannot create utility without kubeconfig")
	}

	return &nodeProblemDetector{
		kubeconfigPath: kubeconfigPath,
		logger:         logger.WithField("cluster-utility", model.NodeProblemDetectorCanonicalName),
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.NodeProblemDetector,
	}, nil
}

func (f *nodeProblemDetector) Destroy() error {
	helm := f.newHelmDeployment(f.logger)
	return helm.Delete()
}

func (f *nodeProblemDetector) Migrate() error {
	return nil
}

func (f *nodeProblemDetector) CreateOrUpgrade() error {
	logger := f.logger.WithField("node-problem-detector-action", "upgrade")
	h := f.newHelmDeployment(logger)

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

func (f *nodeProblemDetector) newHelmDeployment(logger log.FieldLogger) *helmDeployment {
	return newHelmDeployment(
		"deliveryhero/node-problem-detector",
		"node-problem-detector",
		"node-problem-detector",
		f.kubeconfigPath,
		f.desiredVersion,
		defaultHelmDeploymentSetArgument,
		logger,
	)
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
