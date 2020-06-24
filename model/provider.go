// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"fmt"
	"strings"
)

const (
	// ProviderAWS is the cloud provider AWS.
	ProviderAWS = "aws"
)

// CheckProvider normalizes the given provider, returning an error if invalid.
func CheckProvider(provider string) (string, error) {
	provider = strings.ToLower(provider)
	if provider == ProviderAWS {
		return provider, nil
	}

	return provider, fmt.Errorf("unsupported provider %s", provider)
}
