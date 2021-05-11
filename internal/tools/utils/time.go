// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utils

import "time"

// TimeFromMillis converts time in milliseconds to time.Time.
func TimeFromMillis(millis int64) time.Time {
	return time.Unix(0, millis*int64(time.Millisecond))
}

// GetMillis is a convenience method to get milliseconds since epoch.
func GetMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}
