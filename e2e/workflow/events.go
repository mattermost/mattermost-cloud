// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

// +build e2e

package workflow

import "github.com/mattermost/mattermost-cloud/e2e/pkg/eventstest"

func GetExpectedEvents(workflow []*Step) []eventstest.EventOccurrence {
	var events []eventstest.EventOccurrence

	for _, step := range workflow {
		if step.GetExpectedEvents != nil {
			events = append(events, step.GetExpectedEvents()...)
		}
	}

	return events
}
