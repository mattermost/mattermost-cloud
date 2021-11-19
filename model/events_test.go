// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEventDeliveryForEvent(t *testing.T) {
	eventID := "event1"
	desiredDelivery := &EventDelivery{
		ID:      "desired-ed",
		EventID: eventID,
	}

	for _, testCase := range []struct {
		description      string
		deliveries       []*EventDelivery
		expectedDelivery *EventDelivery
		found            bool
	}{
		{
			description: "find first",
			deliveries: []*EventDelivery{
				{ID: "ed1", EventID: "some event"},
				{ID: "ed2", EventID: "other event"},
				desiredDelivery,
				{ID: "ed3", EventID: eventID},
			},
			expectedDelivery: desiredDelivery,
			found:            true,
		},
		{
			description: "do not find",
			deliveries: []*EventDelivery{
				{ID: "ed1", EventID: "some event"},
				{ID: "ed2", EventID: "other event"},
				{ID: "ed3", EventID: "third event"},
			},
			expectedDelivery: nil,
			found:            false,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			delivery, found := EventDeliveryForEvent(eventID, testCase.deliveries)
			assert.Equal(t, testCase.found, found)
			assert.Equal(t, testCase.expectedDelivery, delivery)
		})
	}

}
