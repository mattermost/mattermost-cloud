// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package cloudflare

import (
	"context"

	cf "github.com/cloudflare/cloudflare-go"
)

// MockCloudflare mocks the Cloudflarer  interface
type MockCloudflare struct {
	mockGetZoneID       func(zoneName string) (zoneID string, err error)
	mockCreateDNSRecord func(ctx context.Context, rc *cf.ResourceContainer, params cf.CreateDNSRecordParams) (cf.DNSRecord, error)
	mockDeleteDNSRecord func(ctx context.Context, rc *cf.ResourceContainer, recordID string) error
	mockUpdateDNSRecord func(ctx context.Context, rc *cf.ResourceContainer, params cf.UpdateDNSRecordParams) (cf.DNSRecord, error)
	mockListDNSRecords  func(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error)
}

// MockAWSClient mocks the AWS client interface
type MockAWSClient struct {
	mockGetPublicHostedZoneNames func() []string
}

// GetPublicHostedZoneNames mocks AWS client method
func (a *MockAWSClient) GetPublicHostedZoneNames() []string {
	return a.mockGetPublicHostedZoneNames()
}

// ZoneIDByName mocks the getZoneID
func (c *MockCloudflare) ZoneIDByName(zoneName string) (string, error) {
	return c.mockGetZoneID(zoneName)
}

// DNSRecords mocks cloudflare package same method
func (c *MockCloudflare) ListDNSRecords(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error) {
	return c.mockListDNSRecords(ctx, rc, params)
}

// CreateDNSRecord mocks cloudflare package same method
func (c *MockCloudflare) CreateDNSRecord(ctx context.Context, rc *cf.ResourceContainer, params cf.CreateDNSRecordParams) (cf.DNSRecord, error) {
	return c.mockCreateDNSRecord(ctx, rc, params)
}

// UpdateDNSRecord mocks cloudflare package same method
func (c *MockCloudflare) UpdateDNSRecord(ctx context.Context, rc *cf.ResourceContainer, params cf.UpdateDNSRecordParams) (cf.DNSRecord, error) {
	return c.mockUpdateDNSRecord(ctx, rc, params)
}

// DeleteDNSRecord mocks cloudflare package same method
func (c *MockCloudflare) DeleteDNSRecord(ctx context.Context, rc *cf.ResourceContainer, recordID string) error {
	return c.mockDeleteDNSRecord(ctx, rc, recordID)
}
