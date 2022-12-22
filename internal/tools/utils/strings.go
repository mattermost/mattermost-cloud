// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utils

import (
	"regexp"
	"strings"
)

var rfc1123Characters = regexp.MustCompile(`[^a-z0-9-.]+`)

// SanitizeRFC1123String converts a string to a valid RFC1123 representation, converting all letters to
// lowercase and then revoming all invalid characters.
func SanitizeRFC1123String(input string) string {
	return rfc1123Characters.ReplaceAllString(strings.ToLower(input), "")
}
