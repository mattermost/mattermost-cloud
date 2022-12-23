// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utils

import (
	"regexp"
	"strings"
)

var alphaNumericCharacters = regexp.MustCompile(`[^a-z0-9]+`)

// SanitizeAlphaNumericString converts a string to a valid AlphaNumeric representation, converting all letters to
// lowercase and then removing all invalid characters.
func SanitizeAlphaNumericString(input string) string {
	return alphaNumericCharacters.ReplaceAllString(strings.ToLower(input), "")
}
