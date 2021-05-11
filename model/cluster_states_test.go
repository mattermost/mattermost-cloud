// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
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
