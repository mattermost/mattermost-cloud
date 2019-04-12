package provisioner

import (
	"fmt"
	"strings"
)

// providerAWS is the cloud provider AWS.
const providerAWS = "aws"

func checkProvider(provider string) (string, error) {
	provider = strings.ToLower(provider)
	if provider == providerAWS {
		return provider, nil
	}

	return provider, fmt.Errorf("unsupported provider %s", provider)
}
