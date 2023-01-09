// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
)

// NewAWSConfig retrieves the default AWS configuration from a central place for the SDK v2,
// using a default region if it cannot be loaded.
// To get the order in which the configuration is loaded read the docstring for LoadDefaultConfig
func NewAWSConfig(ctx context.Context) (aws.Config, error) {
	return config.LoadDefaultConfig(
		ctx,
		config.WithDefaultRegion(DefaultAWSRegion),
		config.WithRetryMaxAttempts(DefaultAWSClientRetries),
	)
}

// GetAWSRegion returns the currently setup AWS region with sensible defaults
// for components that only require the region and not a full AWSConfig object
func GetAWSRegion() string {
	region := DefaultAWSRegion
	if regionEnvValue := os.Getenv("AWS_REGION"); regionEnvValue != "" {
		region = regionEnvValue
	}

	return region
}
