package awsv2

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/sirupsen/logrus"
)

// NewConfig loads and returns a new AWS configuration
func NewConfig(logger logrus.FieldLogger) aws.Config {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		logger.Fatalf("Unable to load AWS SDK config, %v", err)
	}

	return cfg
}
