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
