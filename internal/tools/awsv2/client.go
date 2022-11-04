// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package awsv2

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

// awsServices stores the individual services exposes by the AWS SDK
type awsServices struct {
	acm            *acm.Client
	ec2            *ec2.Client
	secretsManager *secretsmanager.Client
}

// Client a client to interact with AWS resources
type Client struct {
	aws *awsServices

	store  model.InstallationDatabaseStoreInterface
	config *config.Config
	logger log.FieldLogger
}

// NewClientWithConfig returns a new instance of Client with a custom configuration.
func NewClientWithConfig(cfg aws.Config, logger log.FieldLogger) (*Client, error) {
	services := awsServices{
		acm: acm.NewFromConfig(cfg),
		ec2: ec2.NewFromConfig(cfg),
	}
	client := &Client{
		logger: logger,
		aws:    &services,
	}

	return client, nil
}

// NewClient returns a new instance of Client
func NewClient(logger log.FieldLogger) (*Client, error) {
	cfg := NewConfig(logger)
	services := awsServices{
		acm: acm.NewFromConfig(cfg),
	}
	client := &Client{
		logger: logger,
		aws:    &services,
	}

	return client, nil
}
