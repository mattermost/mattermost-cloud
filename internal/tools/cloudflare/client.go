// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package cloudflare

import (
	"context"

	cf "github.com/cloudflare/cloudflare-go"
)

// Cloudflarer interface that holds Cloudflare functions
type Cloudflarer interface {
	ZoneIDByName(zoneName string) (string, error)
	ListDNSRecords(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error)
	CreateDNSRecord(ctx context.Context, rc *cf.ResourceContainer, params cf.CreateDNSRecordParams) (cf.DNSRecord, error)
	UpdateDNSRecord(ctx context.Context, rc *cf.ResourceContainer, params cf.UpdateDNSRecordParams) (cf.DNSRecord, error)
	DeleteDNSRecord(ctx context.Context, rc *cf.ResourceContainer, recordID string) error
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
