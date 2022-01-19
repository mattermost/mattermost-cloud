// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//+build e2e

package eventstest

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventsRecorder(t *testing.T) {

	mockEvents := []*model.StateChangeEventPayload{
		{ResourceType: model.TypeCluster, ResourceID: "cluster1"},
		{ResourceType: model.TypeInstallation, ResourceID: "installation"},
		{ResourceType: model.TypeClusterInstallation, ResourceID: "clusterInstallation1"},
	}

	clusterEventOcc := EventOccurrence{ResourceType: model.TypeCluster.String(), ResourceID: "cluster1"}
	installationEventOcc := EventOccurrence{ResourceType: model.TypeInstallation.String(), ResourceID: "installation"}
	clusterInstallationEventOcc := EventOccurrence{ResourceType: model.TypeClusterInstallation.String(), ResourceID: "clusterInstallation1"}

	recorderURL := "http://localhost:11112"

	for _, testCase := range []struct {
		description    string
		recorder       *EventsRecorder
		recordedEvents []EventOccurrence
	}{
		{
			description: "record all and verify",
			recorder:    NewEventsRecorder("test", recorderURL, logrus.New(), RecordAll),
			recordedEvents: []EventOccurrence{
				clusterEventOcc, installationEventOcc, clusterInstallationEventOcc,
			},
		},
		{
			description: "record installation only",
			recorder:    NewEventsRecorder("test", recorderURL, logrus.New(), RecordInstallation),
			recordedEvents: []EventOccurrence{
				installationEventOcc,
			},
		},
		{
			description: "record cluster only",
			recorder:    NewEventsRecorder("test", recorderURL, logrus.New(), RecordCluster),
			recordedEvents: []EventOccurrence{
				clusterEventOcc,
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			err := testCase.recorder.runServer()
			require.NoError(t, err)
			defer testCase.recorder.server.Close()

			for _, mockEve := range mockEvents {
				mockEvent(t, recorderURL, mockEve)
			}

			assert.Equal(t, len(testCase.recordedEvents), len(testCase.recorder.RecordedEvents))

			err = testCase.recorder.VerifyInOrder(testCase.recordedEvents)
			require.NoError(t, err)
		})
	}

	t.Run("verification fail", func(t *testing.T) {
		recorder := NewEventsRecorder("test", recorderURL, logrus.New(), RecordAll)
		err := recorder.runServer()
		require.NoError(t, err)
		defer recorder.server.Close()

		for _, mockEve := range mockEvents {
			mockEvent(t, recorderURL, mockEve)
		}

		incorectOrderOccurences := []EventOccurrence{
			clusterEventOcc, clusterInstallationEventOcc, installationEventOcc,
		}

		err = recorder.VerifyInOrder(incorectOrderOccurences)
		require.Error(t, err)
	})
}

func mockEvent(t *testing.T, url string, eventPayload *model.StateChangeEventPayload) {
	payload, err := json.Marshal(eventPayload)
	require.NoError(t, err)

	_, err = http.Post(url, "application/json", bytes.NewBuffer(payload))
	require.NoError(t, err)
}
