// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
// 

package awsv2

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	log "github.com/sirupsen/logrus"
)

// awsServices stores the individual services exposes by the AWS SDK
type awsServices struct {
	ACM *acm.Client
}

// Client a client to interact with AWS resources
type Client struct {
	aws *awsServices

	config *config.Config
	logger log.FieldLogger
}

// NewClientWithConfig returns a new instance of Client with a custom configuration.
func NewClientWithConfig(cfg aws.Config, logger log.FieldLogger) (*Client, error) {
	services := awsServices{
		ACM: acm.NewFromConfig(cfg),
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
		ACM: acm.NewFromConfig(cfg),
	}
	client := &Client{
		logger: logger,
		aws:    &services,
	}

	return client, nil
}
