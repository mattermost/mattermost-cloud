// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import "time"

// GetMillis is a convenience method to get milliseconds since epoch.
func GetMillis() int64 {
	return GetMillisAtTime(time.Now())
}

// GetMillisAtTime returns millis for a given time.
func GetMillisAtTime(t time.Time) int64 {
	return t.UnixNano() / int64(time.Millisecond)
}

// TimeFromMillis converts time in milliseconds to time.Time.
func TimeFromMillis(millis int64) time.Time {
	return time.Unix(0, millis*int64(time.Millisecond))
}

// DateStringFromMillis returns a standard date string from millis.
func DateStringFromMillis(millis int64) string {
	return TimeFromMillis(millis).Format("Jan 2 2006")
}

// DateTimeStringFromMillis returns a standard complete time string from millis.
func DateTimeStringFromMillis(millis int64) string {
	return TimeFromMillis(millis).Format("2006-01-02 15:04:05 MST")
}

// ElapsedTimeInSeconds returns time in seconds since the provided millis.
func ElapsedTimeInSeconds(millis int64) float64 {
	return float64(GetMillis()-millis) / 1000
}
