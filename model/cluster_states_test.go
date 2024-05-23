// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCluster_ValidTransitionState(t *testing.T) {
	// Couple of tests to verify mechanism is working - we can add more for specific cases
	for _, testCase := range []struct {
		oldState string
		newState string
		isValid  bool
	}{
		{
			oldState: ClusterStateCreationRequested,
			newState: ClusterStateCreationRequested,
			isValid:  true,
		},
		{
			oldState: ClusterStateCreationFailed,
			newState: ClusterStateCreationRequested,
			isValid:  true,
		},
		{
			oldState: ClusterStateStable,
			newState: ClusterStateResizeRequested,
			isValid:  true,
		},
		{
			oldState: ClusterStateResizeRequested,
			newState: ClusterStateUpgradeRequested,
			isValid:  false,
		},
		{
			oldState: ClusterStateProvisioningRequested,
			newState: ClusterStateResizeRequested,
			isValid:  false,
		},
	} {
		t.Run(testCase.oldState+" to "+testCase.newState, func(t *testing.T) {
			cluster := Cluster{State: testCase.oldState}

			isValid := cluster.ValidTransitionState(testCase.newState)
			assert.Equal(t, testCase.isValid, isValid)
		})
	}
}

func TestClusterCanScheduleInstallations(t *testing.T) {
	for _, testCase := range []struct {
		state              string
		allowInstallations bool
		canSchedule        bool
	}{
		{
			state:              ClusterStateCreationRequested,
			allowInstallations: true,
			canSchedule:        false,
		},
		{
			state:              ClusterStateCreationInProgress,
			allowInstallations: true,
			canSchedule:        false,
		},
		{
			state:              ClusterStateDeletionRequested,
			allowInstallations: true,
			canSchedule:        false,
		},
		{
			state:              ClusterStateDeletionFailed,
			allowInstallations: true,
			canSchedule:        false,
		},
		{
			state:              ClusterStateDeleted,
			allowInstallations: true,
			canSchedule:        false,
		},
		{
			state:              ClusterStateProvisionInProgress,
			allowInstallations: false,
			canSchedule:        false,
		},
		{
			state:              ClusterStateProvisionInProgress,
			allowInstallations: true,
			canSchedule:        true,
		},
		{
			state:              ClusterStateResizeRequested,
			allowInstallations: false,
			canSchedule:        false,
		},
		{
			state:              ClusterStateResizeRequested,
			allowInstallations: true,
			canSchedule:        true,
		},
		{
			state:              ClusterStateUpgradeRequested,
			allowInstallations: false,
			canSchedule:        false,
		},
		{
			state:              ClusterStateUpgradeRequested,
			allowInstallations: true,
			canSchedule:        true,
		},
	} {
		t.Run(fmt.Sprintf("state-%s_allow-%v", testCase.state, testCase.allowInstallations), func(t *testing.T) {
			cluster := Cluster{
				State:              testCase.state,
				AllowInstallations: testCase.allowInstallations,
			}
			if testCase.canSchedule {
				assert.NoError(t, cluster.CanScheduleInstallations())
			} else {
				assert.Error(t, cluster.CanScheduleInstallations())
			}
		})
	}
}
