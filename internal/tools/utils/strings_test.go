// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeRFC1123String(t *testing.T) {
	testCases := []struct {
		Input  string
		Output string
	}{
		{
			Input:  "aabbcc",
			Output: "aabbcc",
		},
		{
			Input:  "SpinWick-thingy-123",
			Output: "spinwick-thingy-123",
		},
		{
			Input:  "sup3rR@nD#mS7t))((###-oopp<>../;[",
			Output: "sup3rrndms7t-oopp..",
		},
	}

	for _, testCase := range testCases {
		assert.Equal(t, testCase.Output, SanitizeRFC1123String(testCase.Input))
	}
}
