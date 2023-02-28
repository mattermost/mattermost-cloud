// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package state

import "time"

type E2EState struct {
	ClusterID string
	TestID    string
	StartTime time.Time
	EndTime   time.Time
}

var State E2EState
