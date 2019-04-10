package provisioner

import "fmt"

// providerAWS is the cloud provider AWS.
const providerAWS = "aws"

func checkProvider(provider string) error {
	if provider == providerAWS {
		return nil
	}

	return fmt.Errorf("unsupported provider %s", provider)
}
