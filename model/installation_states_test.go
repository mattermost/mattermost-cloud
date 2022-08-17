// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInstallation_ValidTransitionState(t *testing.T) {

	// Couple of tests to verify mechanism is working - we can add more for specific cases
	for _, testCase := range []struct {
		oldState string
		newState string
		isValid  bool
	}{
		{
			oldState: InstallationStateCreationRequested,
			newState: InstallationStateCreationRequested,
			isValid:  true,
		},
		{
			oldState: InstallationStateCreationFailed,
			newState: InstallationStateCreationRequested,
			isValid:  true,
		},
		{
			oldState: InstallationStateStable,
			newState: InstallationStateHibernationRequested,
			isValid:  true,
		},
		{
			oldState: InstallationStateUpdateInProgress,
			newState: InstallationStateHibernationRequested,
			isValid:  false,
		},
		{
			oldState: InstallationStateHibernating,
			newState: InstallationStateUpdateRequested,
			isValid:  false,
		},
		{
			oldState: InstallationStateStable,
			newState: InstallationStateDeletionRequested,
			isValid:  false,
		},
		{
			oldState: InstallationStateStable,
			newState: InstallationStateDeletionPendingRequested,
			isValid:  true,
		},
		{
			oldState: InstallationStateHibernating,
			newState: InstallationStateDeletionRequested,
			isValid:  false,
		},
		{
			oldState: InstallationStateHibernating,
			newState: InstallationStateDeletionPendingRequested,
			isValid:  true,
		},
		{
			oldState: InstallationStateDeletionPending,
			newState: InstallationStateDeletionRequested,
			isValid:  false,
		},
		{
			oldState: InstallationStateCreationNoCompatibleClusters,
			newState: InstallationStateDeletionRequested,
			isValid:  true,
		},
		{
			oldState: InstallationStateCreationInProgress,
			newState: InstallationStateDeletionRequested,
			isValid:  true,
		},
	} {
		t.Run(testCase.oldState+" to "+testCase.newState, func(t *testing.T) {
			installation := Installation{State: testCase.oldState}

			isValid := installation.ValidTransitionState(testCase.newState)
			assert.Equal(t, testCase.isValid, isValid)
		})
	}
}
