// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

func SToP(s string) *string {
	return &s
}

func IToP(i int64) *int64 {
	return &i
}

func BToP(b bool) *bool {
	return &b
}
