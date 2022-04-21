package cloudflare

import (
	"context"

	cf "github.com/cloudflare/cloudflare-go"
	"github.com/sirupsen/logrus"
)

// MockCloudflare mocks the Cloudflarer  interface
type MockCloudflare struct {
	mockGetZoneID       func(zoneName string) (zoneID string, err error)
	mockGetZoneName     func(zoneNameList []string, customerDNSName string) (zoneName string, found bool)
	mockGetRecordID     func(zoneID, customerDNSName string, logger logrus.FieldLogger) (recordID string, err error)
	mockCreateDNSRecord func(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error)
	mockDeleteDNSRecord func(ctx context.Context, zoneID, recordID string) error
	mockDNSRecords      func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error)
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

func (c *MockCloudflare) getZoneName(zoneNameList []string, customerDNSName string) (zoneName string, found bool) {
	return c.mockGetZoneName(zoneNameList, customerDNSName)
}

func (c *MockCloudflare) getRecordID(zoneID, customerDNSName string, logger logrus.FieldLogger) (recordID string, err error) {
	return c.mockGetRecordID(zoneID, customerDNSName, logger)
}

// DNSRecords mocks cloudflare package same method
func (c *MockCloudflare) DNSRecords(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error) {
	return c.mockDNSRecords(ctx, zoneID, rr)
}

// CreateDNSRecord mocks cloudflare package same method
func (c *MockCloudflare) CreateDNSRecord(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error) {
	return c.mockCreateDNSRecord(ctx, zoneID, rr)
}

// DeleteDNSRecord mocks cloudflare package same method
func (c *MockCloudflare) DeleteDNSRecord(ctx context.Context, zoneID, recordID string) error {
	return c.mockDeleteDNSRecord(ctx, zoneID, recordID)
}
