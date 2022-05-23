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

func TestNewHelmDeploymentWithDefaultConfigurationMetricsServer(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	provisioner := &KopsProvisioner{}
	logger := log.New()
	kops := &kops.Cmd{}
	metricsServer, err := newMetricsServerHandle(&model.HelmUtilityVersion{Chart: "3.8.2"}, &model.Cluster{
		UtilityMetadata: &model.UtilityMetadata{
			ActualVersions: model.UtilityGroupVersions{},
		},
	}, provisioner, kops, logger)
	require.NoError(t, err, "should not error when creating new metrics server handler")
	require.NotNil(t, metricsServer, "metrics server should not be nil")

	helmDeployment := metricsServer.NewHelmDeployment(logger)
	require.NotNil(t, helmDeployment, "helmDeployment should not be nil")
}
