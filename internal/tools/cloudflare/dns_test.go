// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package cloudflare

import (
	"context"
	"testing"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-cloud/internal/testlib"

	"github.com/sirupsen/logrus"

	cf "github.com/cloudflare/cloudflare-go"
	"github.com/stretchr/testify/assert"
)

type mockCloudflare struct {
	mockGetZoneID       func(zoneName string) (zoneID string, err error)
	mockGetZoneName     func(zoneNameList []string, customerDnsName string) (zoneName string, found bool)
	mockGetRecordID     func(zoneID, customerDnsName string, logger logrus.FieldLogger) (recordID string, err error)
	mockCreateDNSRecord func(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error)
	mockDeleteDNSRecord func(ctx context.Context, zoneID, recordID string) error
	mockDNSRecords      func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error)
}

func (c *mockCloudflare) ZoneIDByName(zoneName string) (string, error) {
	return c.mockGetZoneID(zoneName)
}

func (c *mockCloudflare) getZoneName(zoneNameList []string, customerDNSName string) (zoneName string, found bool) {
	return c.mockGetZoneName(zoneNameList, customerDNSName)
}

func (c *mockCloudflare) getRecordId(zoneID, customerDNSName string, logger logrus.FieldLogger) (recordID string, err error) {
	return c.mockGetRecordID(zoneID, customerDNSName, logger)
}

func (c *mockCloudflare) DNSRecords(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error) {
	return c.mockDNSRecords(ctx, zoneID, rr)
}
func (c *mockCloudflare) CreateDNSRecord(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error) {
	return c.mockCreateDNSRecord(ctx, zoneID, rr)
}

func (c *mockCloudflare) DeleteDNSRecord(ctx context.Context, zoneID, recordID string) error {
	return c.mockDeleteDNSRecord(ctx, zoneID, recordID)
}

func TestGetZoneID(t *testing.T) {
	mockCF := &mockCloudflare{}
	samples := []struct {
		description string
		zoneName    string
		setup       func(zoneName string) (zoneID string, err error)
		expected    string
	}{
		{
			description: "return failed and empty string",
			zoneName:    "notexistingdns",
			setup: func(zoneName string) (zoneID string, err error) {
				return "", errors.New("failed")
			},
			expected: "",
		},
		{
			description: "success",
			zoneName:    "existingdns.com",
			setup: func(zoneName string) (zoneID string, err error) {
				return "RANDOMDIDFROMCLOUDFLARE", nil
			},
			expected: "RANDOMDIDFROMCLOUDFLARE",
		},
	}

	for _, s := range samples {
		t.Run(s.description, func(t *testing.T) {
			mockCF.mockGetZoneID = s.setup
			client := NewClientWithToken(mockCF)
			id, _ := client.getZoneID(s.zoneName)
			assert.Equal(t, s.expected, id)
		})
	}

}

func TestGetZoneName(t *testing.T) {
	type Expected struct {
		string
		bool
	}
	mockCF := &mockCloudflare{}
	samples := []struct {
		description     string
		zoneNameList    []string
		customerDNSName string
		setup           func(zoneNameList []string, customerDnsName string) (zoneName string, found bool)
		expected        Expected
	}{
		{
			description:     "success with 1 zone name in the list",
			zoneNameList:    []string{"cloud.mattermost.com"},
			customerDNSName: "customer.cloud.mattermost.com",
			setup: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "cloud.mattermost.com", true
			},
			expected: Expected{"cloud.mattermost.com", true},
		},
		{
			description:     "success with 2 zone name in the list",
			zoneNameList:    []string{"cloud.mattermost.com", "cloud.test.mattermost.com"},
			customerDNSName: "customer.cloud.mattermost.com",
			setup: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "cloud.mattermost.com", true
			},
			expected: Expected{"cloud.mattermost.com", true},
		},
		{
			description:     "failure with 1 zone name in the list",
			zoneNameList:    []string{"cloud.env.mattermost.com"},
			customerDNSName: "customer.cloud.mattermost.com",
			setup: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "", false
			},
			expected: Expected{"", false},
		},
		{
			description:     "failure with 2 zone name in the list",
			zoneNameList:    []string{"cloud.env.mattermost.com", "cloud.test.mattermost.com"},
			customerDNSName: "customer.cloud.mattermost.com",
			setup: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "", false
			},
			expected: Expected{"", false},
		},
		{
			description:     "failure empty zone name in the list",
			zoneNameList:    []string{},
			customerDNSName: "customer.cloud.mattermost.com",
			setup: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "", false
			},
			expected: Expected{"", false},
		},
		{
			description:     "failure empty customer DNS name",
			zoneNameList:    []string{"cloud.env.mattermost.com", "cloud.test.mattermost.com"},
			customerDNSName: "",
			setup: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "", false
			},
			expected: Expected{"", false},
		},
	}

	for _, s := range samples {
		t.Run(s.description, func(t *testing.T) {
			mockCF.mockGetZoneName = s.setup
			client := NewClientWithToken(mockCF)
			name, found := client.getZoneName(s.zoneNameList, s.customerDNSName)
			result := Expected{name, found}
			assert.Equal(t, s.expected, result)
		})
	}

}

func TestGetRecordID(t *testing.T) {
	logger := testlib.MakeLogger(t)
	type Expected struct {
		string
		error
	}
	mockCF := &mockCloudflare{}
	samples := []struct {
		description     string
		zoneID          string
		customerDNSName string
		setup           func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error)
		expected        Expected
	}{
		{
			description:     "success with 1 zone name in the list",
			zoneID:          "THISISAZONEIDFROMCLOUDFLARE",
			customerDNSName: "customer.cloud.mattermost.com",
			setup: func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error) {
				return []cf.DNSRecord{
					{
						ID: "CLOUDFLARERECORDID",
					},
				}, nil
			},
			expected: Expected{"CLOUDFLARERECORDID", nil},
		},
		{
			description:     "non existing zone ID at Cloudflare",
			zoneID:          "NONEXISTINGIDATCLOUDFLARE",
			customerDNSName: "customer.cloud.mattermost.com",
			setup: func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error) {
				return []cf.DNSRecord{}, nil
			},
			expected: Expected{"", nil},
		},
		{
			description:     "error while calling cloudflare API",
			zoneID:          "NONEXISTINGIDATCLOUDFLARE",
			customerDNSName: "customer.cloud.mattermost.com",
			setup: func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error) {
				return []cf.DNSRecord{}, errors.New("Cloudflare API error")
			},
			expected: Expected{"", errors.New("Cloudflare API error")},
		},
	}

	for _, s := range samples {
		t.Run(s.description, func(t *testing.T) {
			mockCF.mockDNSRecords = s.setup
			client := NewClientWithToken(mockCF)
			name, err := client.getRecordID(s.zoneID, s.customerDNSName, logger)
			result := Expected{name, err}
			if err != nil {
				assert.EqualError(t, s.expected, err.Error())
				return
			}
			assert.Equal(t, s.expected, result)
		})
	}

}

func TestCreateDNSRecord(t *testing.T) {
	logger := testlib.MakeLogger(t)

	mockCF := &mockCloudflare{}
	samples := []struct {
		description     string
		customerDNSName string
		zoneNameList    []string
		dnsEndpoints    []string
		setupName       func(zoneNameList []string, customerDnsName string) (zoneName string, found bool)
		setupID         func(zoneName string) (zoneID string, err error)
		setupDNS        func(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error)
		expected        error
	}{
		{
			description:     "success with 1 zone name in the list",
			customerDNSName: "customer.cloud.mattermost.com",
			zoneNameList:    []string{"cloud.mattermost.com"},
			dnsEndpoints:    []string{"load.balancer.endpoint"},
			setupName: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "cloud.mattermost.com", true
			},
			setupID: func(zoneName string) (zoneID string, err error) {
				return "RANDOMDIDFROMCLOUDFLARE", nil
			},
			setupDNS: func(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error) {
				return &cf.DNSRecordResponse{
					Result: cf.DNSRecord{
						ID: "CLOUDFLARERECORDID",
					},
				}, nil
			},
			expected: nil,
		},
		{
			description:     "success with 2 zone name, 1 dns endpoints in the list",
			customerDNSName: "customer.cloud.mattermost.com",
			zoneNameList:    []string{"cloud.mattermost.com", "cloud.test.mattermost.com"},
			dnsEndpoints:    []string{"load.balancer.endpoint"},
			setupName: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "cloud.mattermost.com", true
			},
			setupID: func(zoneName string) (zoneID string, err error) {
				return "RANDOMDIDFROMCLOUDFLARE", nil
			},
			setupDNS: func(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error) {
				return &cf.DNSRecordResponse{
					Result: cf.DNSRecord{
						ID: "CLOUDFLARERECORDID",
					},
				}, nil
			},
			expected: nil,
		},
		{
			description:     "success with 2 zone name, 2 dns endpoints in the list",
			customerDNSName: "customer.cloud.mattermost.com",
			zoneNameList:    []string{"cloud.mattermost.com", "cloud.test.mattermost.com"},
			dnsEndpoints:    []string{"load.balancer.endpoint", "second.load.balancer.endpoint"},
			setupName: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "cloud.mattermost.com", true
			},
			setupID: func(zoneName string) (zoneID string, err error) {
				return "RANDOMDIDFROMCLOUDFLARE", nil
			},
			setupDNS: func(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error) {
				return &cf.DNSRecordResponse{
					Result: cf.DNSRecord{
						ID: "CLOUDFLARERECORDID",
					},
				}, nil
			},
			expected: nil,
		},
		{
			description:     "failure with empty zone name",
			customerDNSName: "",
			zoneNameList:    []string{},
			dnsEndpoints:    []string{"load.balancer.endpoint"},
			setupName: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "", false
			},
			setupID: func(zoneName string) (zoneID string, err error) {
				return "", nil
			},
			setupDNS: func(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error) {
				return &cf.DNSRecordResponse{}, nil
			},
			expected: errors.New("hosted zone for \"\" domain name not found"),
		},
		{
			description:     "failure with no DNS endpoints",
			customerDNSName: "customer.cloud.mattermost.com",
			zoneNameList:    []string{"cloud.mattermost.com"},
			dnsEndpoints:    []string{},
			setupName: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "", false
			},
			setupID: func(zoneName string) (zoneID string, err error) {
				return "", nil
			},
			setupDNS: func(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error) {
				return &cf.DNSRecordResponse{}, nil
			},
			expected: errors.New("no DNS endpoints provided for Cloudflare creation request"),
		},
		{
			description:     "failure with empty string DNS endpoint",
			customerDNSName: "customer.cloud.mattermost.com",
			zoneNameList:    []string{"cloud.mattermost.com"},
			dnsEndpoints:    []string{""},
			setupName: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "", false
			},
			setupID: func(zoneName string) (zoneID string, err error) {
				return "", nil
			},
			setupDNS: func(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error) {
				return &cf.DNSRecordResponse{}, nil
			},
			expected: errors.New("DNS endpoint was an empty string"),
		},
		{
			description:     "failure with zone ID fetching",
			customerDNSName: "customer.cloud.mattermost.com",
			zoneNameList:    []string{"cloud.mattermost.com"},
			dnsEndpoints:    []string{"load.balancer.endpoint"},
			setupName: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "", false
			},
			setupID: func(zoneName string) (zoneID string, err error) {
				return "", nil
			},
			setupDNS: func(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error) {
				return &cf.DNSRecordResponse{}, nil
			},
			expected: errors.New("failed to fetch Zone ID from Cloudflare"),
		},
	}

	for _, s := range samples {
		t.Run(s.description, func(t *testing.T) {
			mockCF.mockGetZoneName = s.setupName
			mockCF.mockGetZoneID = s.setupID
			mockCF.mockCreateDNSRecord = s.setupDNS
			client := NewClientWithToken(mockCF)
			err := client.CreateDNSRecord(s.customerDNSName, s.zoneNameList, s.dnsEndpoints, logger)
			if err != nil {
				assert.EqualError(t, s.expected, err.Error())
			}
		})
	}
}

func TestDeleteDNSRecord(t *testing.T) {
	logger := testlib.MakeLogger(t)

	mockCF := &mockCloudflare{}
	samples := []struct {
		description          string
		customerDnsName      string
		zoneNameList         []string
		zoneID               string
		setupName            func(zoneNameList []string, customerDnsName string) (zoneName string, found bool)
		setupZoneID          func(zoneName string) (zoneID string, err error)
		setupRecordID        func(zoneID, customerDnsName string, logger logrus.FieldLogger) (recordID string, err error)
		setupDeleteDNSRecord func(ctx context.Context, zoneID, recordID string) error
		setupDNSRecord       func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error)
		expected             error
	}{
		{
			description:     "success path",
			customerDnsName: "customer.cloud.mattermost.com",
			zoneNameList:    []string{"cloud.mattermost.com"},
			zoneID:          "RANDOMDIDFROMCLOUDFLARE",
			setupName: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "cloud.mattermost.com", true
			},
			setupZoneID: func(zoneName string) (zoneID string, err error) {
				return "RANDOMDIDFROMCLOUDFLARE", nil
			},
			setupRecordID: func(zoneID, customerDnsName string, logger logrus.FieldLogger) (recordID string, err error) {
				return "RANDOMRECORDIDFROMCLOUDFLARE", nil
			},
			setupDeleteDNSRecord: func(ctx context.Context, zoneID, recordID string) error {
				return nil
			},
			setupDNSRecord: func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error) {
				return []cf.DNSRecord{
					{
						ID: "CLOUDFLARERECORDID",
					},
				}, nil
			},
			expected: nil,
		},
		{
			description:     "success with 2 zone name, 1 dns endpoints in the list",
			customerDnsName: "customer.cloud.mattermost.com",
			zoneNameList:    []string{"cloud.mattermost.com", "cloud.test.mattermost.com"},
			zoneID:          "RANDOMDIDFROMCLOUDFLARE",
			setupName: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "cloud.mattermost.com", true
			},
			setupZoneID: func(zoneName string) (zoneID string, err error) {
				return "RANDOMDIDFROMCLOUDFLARE", nil
			},
			setupRecordID: func(zoneID, customerDnsName string, logger logrus.FieldLogger) (recordID string, err error) {
				return "RANDOMRECORDIDFROMCLOUDFLARE", nil
			},
			setupDeleteDNSRecord: func(ctx context.Context, zoneID, recordID string) error {
				return nil
			},
			setupDNSRecord: func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error) {
				return []cf.DNSRecord{
					{
						ID: "CLOUDFLARERECORDID",
					},
				}, nil
			},
			expected: nil,
		},
		{
			description:     "failure to get zone ID",
			customerDnsName: "customer.cloud.mattermost.com",
			zoneNameList:    []string{"cloud.mattermost.com"},
			zoneID:          "RANDOMDIDFROMCLOUDFLARE",
			setupName: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "cloud.mattermost.com", true
			},
			setupZoneID: func(zoneName string) (zoneID string, err error) {
				return "", errors.New("failed to get zone ID")
			},
			setupRecordID: func(zoneID, customerDnsName string, logger logrus.FieldLogger) (recordID string, err error) {
				return "", nil
			},
			setupDeleteDNSRecord: func(ctx context.Context, zoneID, recordID string) error {
				return nil
			},
			setupDNSRecord: func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error) {
				return []cf.DNSRecord{}, nil
			},
			expected: errors.New("failed to get zone ID"),
		},
		{
			description:     "failure to get record ID",
			customerDnsName: "customer.cloud.mattermost.com",
			zoneNameList:    []string{"cloud.mattermost.com"},
			zoneID:          "RANDOMDIDFROMCLOUDFLARE",
			setupName: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "cloud.mattermost.com", true
			},
			setupZoneID: func(zoneName string) (zoneID string, err error) {
				return "RANDOMDIDFROMCLOUDFLARE", nil
			},
			setupRecordID: func(zoneID, customerDnsName string, logger logrus.FieldLogger) (recordID string, err error) {
				return "", errors.New("failed to get record ID")
			},
			setupDeleteDNSRecord: func(ctx context.Context, zoneID, recordID string) error {
				return nil
			},
			setupDNSRecord: func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error) {
				return []cf.DNSRecord{}, nil
			},
			expected: errors.New("failed to get record ID"),
		},
		{
			description:     "failure to delete DNS record",
			customerDnsName: "customer.cloud.mattermost.com",
			zoneNameList:    []string{"cloud.mattermost.com"},
			zoneID:          "RANDOMDIDFROMCLOUDFLARE",
			setupName: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "cloud.mattermost.com", true
			},
			setupZoneID: func(zoneName string) (zoneID string, err error) {
				return "RANDOMDIDFROMCLOUDFLARE", nil
			},
			setupRecordID: func(zoneID, customerDnsName string, logger logrus.FieldLogger) (recordID string, err error) {
				return "RANDOMRECORDIDFROMCLOUDFLARE", nil
			},
			setupDeleteDNSRecord: func(ctx context.Context, zoneID, recordID string) error {
				return errors.New("failed to delete DNS record")
			},
			setupDNSRecord: func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error) {
				return []cf.DNSRecord{}, nil
			},
			expected: errors.New("failed to delete DNS record"),
		},
	}

	for _, s := range samples {
		t.Run(s.description, func(t *testing.T) {
			mockCF.mockGetZoneName = s.setupName
			mockCF.mockGetZoneID = s.setupZoneID
			mockCF.mockGetRecordID = s.setupRecordID
			mockCF.mockDeleteDNSRecord = s.setupDeleteDNSRecord
			mockCF.mockDNSRecords = s.setupDNSRecord
			client := NewClientWithToken(mockCF)
			err := client.DeleteDNSRecord(s.customerDnsName, s.zoneNameList, logger)
			if err != nil {
				assert.EqualError(t, s.expected, err.Error())
			}
		})
	}
}
