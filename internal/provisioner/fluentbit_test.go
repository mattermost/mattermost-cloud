// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"errors"
	"testing"

	mocks "github.com/mattermost/mattermost-cloud/internal/mocks/aws-tools"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"

	"github.com/golang/mock/gomock"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewHelmDeploymentWithAuditLogsConfiguration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	provisioner := &KopsProvisioner{}
	logger := log.New()
	awsClient := mocks.NewMockAWS(ctrl)
	awsClient.EXPECT().
		GetPrivateZoneDomainName(gomock.Eq(logger)).
		Return("mockDns", nil).
		AnyTimes()
	awsClient.EXPECT().
		GetPrivateZoneIDForDefaultTag(gomock.Eq(logger)).
		Return("mockZone", nil).
		AnyTimes()
	expectedTag := &aws.Tag{Key: "AuditLogsCoreSecurity", Value: "expectedURL:12345"}
	awsClient.EXPECT().
		GetTagByKeyAndZoneID(gomock.Eq("tag:AuditLogsCoreSecurity"), gomock.Eq("mockZone"), gomock.Eq(logger)).
		Return(expectedTag, nil).
		AnyTimes()

	kops := &kops.Cmd{}
	fluentbit, err := newFluentbitHandle("1.2.3", provisioner, awsClient, kops, logger)
	require.NoError(t, err, "should not error when creating new fluentbit handler")
	require.NotNil(t, fluentbit, "fluentbit should not be nil")

	helmDeployment := fluentbit.NewHelmDeployment(logger)
	require.NotNil(t, helmDeployment, "helmDeployment should not be nil")
	assert.Equal(t, "backend.es.host=elasticsearch.mockDns,rawConfig=\n@INCLUDE fluent-bit-service.conf\n@INCLUDE fluent-bit-input.conf\n@INCLUDE fluent-bit-filter.conf\n@INCLUDE fluent-bit-output.conf\n[OUTPUT]\n\tName  forward\n\tMatch  *\n\tHost  expectedURL\n\tPort  12345\n\ttls  On\n\ttls.verify  Off\n", helmDeployment.setArgument)
}

func TestNewHelmDeploymentWithDefaultConfiguration(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	provisioner := &KopsProvisioner{}
	logger := log.New()
	awsClient := mocks.NewMockAWS(ctrl)
	awsClient.EXPECT().
		GetPrivateZoneDomainName(gomock.Eq(logger)).
		Return("mockDns", nil).
		AnyTimes()
	awsClient.EXPECT().
		GetPrivateZoneIDForDefaultTag(gomock.Eq(logger)).
		Return("mockZone", nil).
		AnyTimes()
	expectedTag := &aws.Tag{Key: "MattermostCloudDNS", Value: "private"}
	awsClient.EXPECT().
		GetTagByKeyAndZoneID(gomock.Eq("tag:AuditLogsCoreSecurity"), gomock.Eq("mockZone"), gomock.Eq(logger)).
		Return(expectedTag, nil).
		AnyTimes()

	kops := &kops.Cmd{}
	fluentbit, err := newFluentbitHandle("1.2.3", provisioner, awsClient, kops, logger)
	require.NoError(t, err, "should not error when creating new fluentbit handler")
	require.NotNil(t, fluentbit, "fluentbit should not be nil")

	helmDeployment := fluentbit.NewHelmDeployment(logger)
	require.NotNil(t, helmDeployment, "helmDeployment should not be nil")
	assert.Equal(t, "backend.es.host=elasticsearch.mockDns,rawConfig=\n@INCLUDE fluent-bit-service.conf\n@INCLUDE fluent-bit-input.conf\n@INCLUDE fluent-bit-filter.conf\n@INCLUDE fluent-bit-output.conf\n\n", helmDeployment.setArgument)
}

func TestNewHelmDeploymentWithZoneIDError(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	provisioner := &KopsProvisioner{}
	logger := log.New()
	awsClient := mocks.NewMockAWS(ctrl)

	awsClient.EXPECT().
		GetPrivateZoneDomainName(gomock.Eq(logger)).
		Return("mockDns", nil).
		AnyTimes()
	err1 := errors.New("Mock error expected from func GetPrivateZoneIDForDefaultTag")
	awsClient.EXPECT().
		GetPrivateZoneIDForDefaultTag(gomock.Eq(logger)).
		Return("", err1).
		AnyTimes()

	kops := &kops.Cmd{}
	fluentbit, err := newFluentbitHandle("1.2.3", provisioner, awsClient, kops, logger)
	require.NoError(t, err, "should not error when creating new fluentbit handler")
	require.NotNil(t, fluentbit, "fluentbit should not be nil")

	helmDeployment := fluentbit.NewHelmDeployment(logger)
	require.NotNil(t, helmDeployment, "helmDeployment should not be nil")
	assert.Equal(t, "backend.es.host=elasticsearch.mockDns,rawConfig=\n@INCLUDE fluent-bit-service.conf\n@INCLUDE fluent-bit-input.conf\n@INCLUDE fluent-bit-filter.conf\n@INCLUDE fluent-bit-output.conf\n\n", helmDeployment.setArgument)
}

func TestNewHelmDeploymentWithoutFindingAuditTag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	provisioner := &KopsProvisioner{}
	logger := log.New()
	awsClient := mocks.NewMockAWS(ctrl)
	awsClient.EXPECT().
		GetPrivateZoneDomainName(gomock.Eq(logger)).
		Return("mockDns", nil).
		AnyTimes()
	awsClient.EXPECT().
		GetPrivateZoneIDForDefaultTag(gomock.Eq(logger)).
		Return("mockZone", nil).
		AnyTimes()
	expectedTag := &aws.Tag{}
	err1 := errors.New("Mock error expected from func GetTagByKeyAndZoneID")
	awsClient.EXPECT().
		GetTagByKeyAndZoneID(gomock.Eq("tag:AuditLogsCoreSecurity"), gomock.Eq("mockZone"), gomock.Eq(logger)).
		Return(expectedTag, err1).
		AnyTimes()

	kops := &kops.Cmd{}
	fluentbit, err := newFluentbitHandle("1.2.3", provisioner, awsClient, kops, logger)
	require.NoError(t, err, "should not error when creating new fluentbit handler")
	require.NotNil(t, fluentbit, "fluentbit should not be nil")

	helmDeployment := fluentbit.NewHelmDeployment(logger)
	require.NotNil(t, helmDeployment, "helmDeployment should not be nil")
	assert.Equal(t, "backend.es.host=elasticsearch.mockDns,rawConfig=\n@INCLUDE fluent-bit-service.conf\n@INCLUDE fluent-bit-input.conf\n@INCLUDE fluent-bit-filter.conf\n@INCLUDE fluent-bit-output.conf\n\n", helmDeployment.setArgument)
}

func TestNewHelmDeploymentWithNillTag(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	provisioner := &KopsProvisioner{}
	logger := log.New()
	awsClient := mocks.NewMockAWS(ctrl)
	awsClient.EXPECT().
		GetPrivateZoneDomainName(gomock.Eq(logger)).
		Return("mockDns", nil).
		AnyTimes()
	awsClient.EXPECT().
		GetPrivateZoneIDForDefaultTag(gomock.Eq(logger)).
		Return("mockZone", nil).
		AnyTimes()

	awsClient.EXPECT().
		GetTagByKeyAndZoneID(gomock.Eq("tag:AuditLogsCoreSecurity"), gomock.Eq("mockZone"), gomock.Eq(logger)).
		Return(nil, nil).
		AnyTimes()

	kops := &kops.Cmd{}
	fluentbit, err := newFluentbitHandle("1.2.3", provisioner, awsClient, kops, logger)
	require.NoError(t, err, "should not error when creating new fluentbit handler")
	require.NotNil(t, fluentbit, "fluentbit should not be nil")

	helmDeployment := fluentbit.NewHelmDeployment(logger)
	require.NotNil(t, helmDeployment, "helmDeployment should not be nil")
	assert.Equal(t, "backend.es.host=elasticsearch.mockDns,rawConfig=\n@INCLUDE fluent-bit-service.conf\n@INCLUDE fluent-bit-input.conf\n@INCLUDE fluent-bit-filter.conf\n@INCLUDE fluent-bit-output.conf\n\n", helmDeployment.setArgument)
}
