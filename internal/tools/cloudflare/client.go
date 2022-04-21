// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package cloudflare

import (
	"context"

	cf "github.com/cloudflare/cloudflare-go"
	"github.com/sirupsen/logrus"
)

// Cloudflarer interface that holds Cloudflare functions
type Cloudflarer interface {
	ZoneIDByName(zoneName string) (string, error)
	DNSRecords(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error)
	CreateDNSRecord(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error)
	DeleteDNSRecord(ctx context.Context, zoneID, recordID string) error
}

// AWSClient interface that holds AWS client function
type AWSClient interface {
	GetPublicHostedZoneNames() []string
}

// Client is a wrapper on to of Cloudflare library client.
type Client struct {
	cfClient Cloudflarer
	aws      AWSClient
}

// NewClientWithToken creates a new client that can be used to run the other functions.
func NewClientWithToken(client Cloudflarer, aws AWSClient) *Client {
	return &Client{
		cfClient: client,
		aws:      aws,
	}
}

// NoopCloudflarer is used as a dummy Cloudflarer interface
type NoopCloudflarer struct{}

// NoopClient returns an empty noopCloudflarer struct
func NoopClient() *NoopCloudflarer {
	return &NoopCloudflarer{}
}

// CreateDNSRecords returns an empty dummy func for noopCloudflarer
func (*NoopCloudflarer) CreateDNSRecords(_ []string, _ []string, logger logrus.FieldLogger) error {
	logger.Debug("Using noop Cloudflare client, CreateDNSRecords function")
	return nil
}

// DeleteDNSRecords returns an empty dummy func for noopCloudflarer
func (*NoopCloudflarer) DeleteDNSRecords(_ []string, logger logrus.FieldLogger) error {
	logger.Debug("Using noop Cloudflare client, DeleteDNSRecords function")
	return nil
}
