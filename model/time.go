// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import "time"

// GetMillis is a convenience method to get milliseconds since epoch.
func GetMillis() int64 {
	return time.Now().UnixNano() / int64(time.Millisecond)
}

// TimeFromMillis converts time in milliseconds to time.Time.
func TimeFromMillis(millis int64) time.Time {
	return time.Unix(0, millis*int64(time.Millisecond))
}

// ElapsedTimeInSeconds returns time in seconds since the provided millis.
func ElapsedTimeInSeconds(millis int64) float64 {
	return float64(GetMillis()-millis) / 1000
}
