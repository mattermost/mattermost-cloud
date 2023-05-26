// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utility

import (
	"testing"

	"github.com/golang/mock/gomock"
	mocks "github.com/mattermost/mattermost-cloud/internal/mocks/aws-tools"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func TestNewHelmDeploymentWithDefaultConfiguration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := log.New()
	awsClient := mocks.NewMockAWS(ctrl)
	fluentbit := newFluentbitHandle(&model.Cluster{
		UtilityMetadata: &model.UtilityMetadata{
			ActualVersions: model.UtilityGroupVersions{},
		},
	}, &model.HelmUtilityVersion{Chart: "1.2.3"}, "kubeconfig", awsClient, logger)
	require.NoError(t, fluentbit.validate(), "should not error when creating new fluentbit handler")
	require.NotNil(t, fluentbit, "fluentbit should not be nil")

	helmDeployment := fluentbit.newHelmDeployment()
	require.NotNil(t, helmDeployment, "helmDeployment should not be nil")
}
