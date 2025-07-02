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

type nodeProblemDetector struct {
	kubeconfigPath string
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func newNodeProblemDetectorOrUnmanagedHandle(cluster *model.Cluster, kubeconfigPath string, awsClient aws.AWS, logger log.FieldLogger) (Utility, error) {
	desired := cluster.DesiredUtilityVersion(model.NodeProblemDetectorCanonicalName)
	actual := cluster.ActualUtilityVersion(model.NodeProblemDetectorCanonicalName)

	if model.UtilityIsUnmanaged(desired, actual) {
		return newUnmanagedHandle(model.NodeProblemDetectorCanonicalName, kubeconfigPath, []string{}, cluster, awsClient, logger), nil
	}

	nodeProblemDetector := newNodeProblemDetectorHandle(desired, cluster, kubeconfigPath, logger)
	err := nodeProblemDetector.validate()
	if err != nil {
		return nil, errors.Wrap(err, "node problem detector utility config is invalid")
	}

	return nodeProblemDetector, nil
}

func newNodeProblemDetectorHandle(desiredVersion *model.HelmUtilityVersion, cluster *model.Cluster, kubeconfigPath string, logger log.FieldLogger) *nodeProblemDetector {
	return &nodeProblemDetector{
		kubeconfigPath: kubeconfigPath,
		logger:         logger.WithField("cluster-utility", model.NodeProblemDetectorCanonicalName),
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.NodeProblemDetector,
	}
}

func (n *nodeProblemDetector) validate() error {
	if n.kubeconfigPath == "" {
		return errors.New("kubeconfig path cannot be empty")
	}

	return nil
}

func (n *nodeProblemDetector) Destroy() error {
	helm := n.newHelmDeployment(n.logger)
	return helm.Delete()
}

func (n *nodeProblemDetector) Migrate() error {
	return nil
}

func (n *nodeProblemDetector) CreateOrUpgrade() error {
	logger := n.logger.WithField("node-problem-detector-action", "upgrade")
	h := n.newHelmDeployment(logger)

	err := h.Update()
	if err != nil {
		return err
	}

	err = n.updateVersion(h)
	return err
}

func (n *nodeProblemDetector) DesiredVersion() *model.HelmUtilityVersion {
	return n.desiredVersion
}

func (n *nodeProblemDetector) ActualVersion() *model.HelmUtilityVersion {
	if n.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(n.actualVersion.Version(), "node-problem-detector-"),
		ValuesPath: n.actualVersion.Values(),
	}
}

func (n *nodeProblemDetector) Name() string {
	return model.NodeProblemDetectorCanonicalName
}

func (n *nodeProblemDetector) newHelmDeployment(logger log.FieldLogger) *helmDeployment {
	return newHelmDeployment(
		"deliveryhero/node-problem-detector",
		"node-problem-detector",
		"node-problem-detector",
		n.kubeconfigPath,
		n.desiredVersion,
		defaultHelmDeploymentSetArgument,
		logger,
	)
}

func (n *nodeProblemDetector) ValuesPath() string {
	if n.desiredVersion == nil {
		return ""
	}
	return n.desiredVersion.Values()
}

func (n *nodeProblemDetector) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	n.actualVersion = actualVersion
	return nil
}
