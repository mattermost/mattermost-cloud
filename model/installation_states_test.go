package model

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInstallation_ValidTransitionState(t *testing.T) {

	// Couple of tests to verify mechanism is working - we can add more for specific cases
	for _, testCase := range []struct{
	    oldState string
	    newState string
	    isValid bool
	}{
	    {
			oldState: InstallationStateCreationRequested,
			newState: InstallationStateCreationRequested,
			isValid: true,
	    },
	    {
			oldState: InstallationStateCreationFailed,
			newState: InstallationStateCreationRequested,
			isValid: true,
	    },
	    {
			oldState: InstallationStateStable,
			newState: InstallationStateHibernationRequested,
			isValid: true,
	    },
	    {
			oldState: InstallationStateUpdateInProgress,
			newState: InstallationStateHibernationRequested,
			isValid: false,
	    },
	    {
			oldState: InstallationStateHibernating,
			newState: InstallationStateUpdateRequested,
			isValid: false,
	    },
	} {
	    t.Run(testCase.oldState + " to " +testCase.newState , func(t *testing.T) {
			installation := Installation{State: testCase.oldState}

			isValid := installation.ValidTransitionState(testCase.newState)
			assert.Equal(t, testCase.isValid, isValid)
	    })
	}
}
