// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"

	"github.com/golang/mock/gomock"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestNewHelmDeploymentWithDefaultConfigurationNodeProblemDetector(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	provisioner := &KopsProvisioner{}
	logger := log.New()
	kops := &kops.Cmd{}
	nodeProblemDetector, err := newNodeProblemDetectorHandle(&model.HelmUtilityVersion{Chart: "2.0.5"}, provisioner, kops, logger)
	require.NoError(t, err, "should not error when creating new node-problem-detector handler")
	require.NotNil(t, nodeProblemDetector, "node-problem-detector should not be nil")

	helmDeployment := nodeProblemDetector.NewHelmDeployment(logger)
	require.NotNil(t, helmDeployment, "helmDeployment should not be nil")
}
