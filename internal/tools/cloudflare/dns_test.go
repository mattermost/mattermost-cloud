// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package cloudflare

import (
	"context"
	"testing"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-cloud/internal/testlib"

	cf "github.com/cloudflare/cloudflare-go"
	"github.com/stretchr/testify/assert"
)

func TestGetZoneID(t *testing.T) {
	mockCF := &MockCloudflare{}
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
			client := NewClientWithToken(mockCF, nil)
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
	mockCF := &MockCloudflare{}
	samples := []struct {
		description     string
		zoneNameList    []string
		customerDNSName string
		expected        Expected
	}{
		{
			description:     "success with 1 zone name in the list",
			zoneNameList:    []string{"cloud.mattermost.com"},
			customerDNSName: "customer.cloud.mattermost.com",
			expected: Expected{"cloud.mattermost.com", true},
		},
		{
			description:     "success with 2 zone name in the list",
			zoneNameList:    []string{"cloud.mattermost.com", "cloud.test.mattermost.com"},
			customerDNSName: "customer.cloud.mattermost.com",
			expected: Expected{"cloud.mattermost.com", true},
		},
		{
			description:     "failure with 1 zone name in the list",
			zoneNameList:    []string{"cloud.env.mattermost.com"},
			customerDNSName: "customer.cloud.mattermost.com",
			expected: Expected{"", false},
		},
		{
			description:     "failure with 2 zone name in the list",
			zoneNameList:    []string{"cloud.env.mattermost.com", "cloud.test.mattermost.com"},
			customerDNSName: "customer.cloud.mattermost.com",
			expected: Expected{"", false},
		},
		{
			description:     "failure empty zone name in the list",
			zoneNameList:    []string{},
			customerDNSName: "customer.cloud.mattermost.com",
			expected: Expected{"", false},
		},
		{
			description:     "failure empty customer DNS name",
			zoneNameList:    []string{"cloud.env.mattermost.com", "cloud.test.mattermost.com"},
			customerDNSName: "",
			expected: Expected{"", false},
		},
	}

	for _, s := range samples {
		t.Run(s.description, func(t *testing.T) {
			client := NewClientWithToken(mockCF, nil)
			name, found := client.getZoneName(s.zoneNameList, s.customerDNSName)
			result := Expected{name, found}
			assert.Equal(t, s.expected, result)
		})
	}

}

func TestGetRecordIDs(t *testing.T) {
	logger := testlib.MakeLogger(t)
	type Expected struct {
		ids []string
		err error
	}
	mockCF := &MockCloudflare{}
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
			expected: Expected{[]string{"CLOUDFLARERECORDID"}, nil},
		},
		{
			description:     "success with multiple ids",
			zoneID:          "THISISAZONEIDFROMCLOUDFLARE",
			customerDNSName: "customer.cloud.mattermost.com",
			setup: func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error) {
				return []cf.DNSRecord{
					{ID: "CLOUDFLARERECORDID"},
					{ID: "CLOUDFLARERECORDID2"},
				}, nil
			},
			expected: Expected{[]string{"CLOUDFLARERECORDID","CLOUDFLARERECORDID2"}, nil},
		},
		{
			description:     "non existing zone ID at Cloudflare",
			zoneID:          "NONEXISTINGIDATCLOUDFLARE",
			customerDNSName: "customer.cloud.mattermost.com",
			setup: func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error) {
				return []cf.DNSRecord{}, nil
			},
			expected: Expected{nil, nil},
		},
		{
			description:     "error while calling cloudflare API",
			zoneID:          "NONEXISTINGIDATCLOUDFLARE",
			customerDNSName: "customer.cloud.mattermost.com",
			setup: func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error) {
				return []cf.DNSRecord{}, errors.New("Cloudflare API error")
			},
			expected: Expected{nil, errors.New("failed to get DNS Record ID from Cloudflare: Cloudflare API error")},
		},
	}

	for _, s := range samples {
		t.Run(s.description, func(t *testing.T) {
			mockCF.mockDNSRecords = s.setup
			client := NewClientWithToken(mockCF, nil)
			ids, err := client.getRecordIDs(s.zoneID, s.customerDNSName, logger)
			result := Expected{ids, err}
			if err != nil {
				assert.EqualError(t, s.expected.err, err.Error())
				return
			}
			assert.Equal(t, s.expected, result)
		})
	}

}

func TestCreateDNSRecord(t *testing.T) {
	logger := testlib.MakeLogger(t)

	mockCF := &MockCloudflare{}
	MockAWS := &MockAWSClient{}
	samples := []struct {
		description     string
		customerDNSName []string
		dnsEndpoints    []string
		setupID         func(zoneName string) (zoneID string, err error)
		setupDNS        func(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error)
		awsZoneNameList func() []string
		expected        error
	}{
		{
			description:     "success with 1 zone name in the list",
			customerDNSName: []string{"customer.cloud.mattermost.com"},
			dnsEndpoints:    []string{"load.balancer.endpoint"},
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
			awsZoneNameList: func() []string {
				return []string{"cloud.mattermost.com"}
			},
			expected: nil,
		},
		{
			description:     "success with 2 zone name, 1 dns endpoints in the list",
			customerDNSName: []string{"customer.cloud.mattermost.com"},
			dnsEndpoints:    []string{"load.balancer.endpoint"},
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
			awsZoneNameList: func() []string {
				return []string{"cloud.mattermost.com", "cloud.test.mattermost.com"}
			},
			expected: nil,
		},
		// TODO: success with multiple DNS names
		{
			description:     "success with multiple domain names",
			customerDNSName: []string{"customer.cloud.mattermost.com", "customer.cloud.mattermost.io"},
			dnsEndpoints:    []string{"load.balancer.endpoint"},
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
			awsZoneNameList: func() []string {
				return []string{"cloud.mattermost.com", "cloud.mattermost.io"}
			},
			expected: nil,
		},
		{
			description:     "failure with multiple dns endpoints in the list",
			customerDNSName: []string{"customer.cloud.mattermost.com"},
			dnsEndpoints:    []string{"load.balancer.endpoint", "second.load.balancer.endpoint"},
			expected: errors.New("creating record for more than one endpoint not supported"),
		},
		{
			description:     "failure with no domain names",
			customerDNSName: nil,
			dnsEndpoints:    []string{"load.balancer.endpoint"},
			setupID:         nil,
			setupDNS:        nil,
			awsZoneNameList: nil,
			expected:        errors.New("no domain names provided"),
		},
		{
			description:     "failure with empty zone name",
			customerDNSName: []string{""},
			dnsEndpoints:    []string{"load.balancer.endpoint"},
			setupID: func(zoneName string) (zoneID string, err error) {
				return "", nil
			},
			setupDNS: func(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error) {
				return &cf.DNSRecordResponse{}, nil
			},
			awsZoneNameList: func() []string {
				return []string{""}
			},
			expected: errors.New("hosted zone for \"\" domain name not found"),
		},
		{
			description:     "failure with no DNS endpoints",
			customerDNSName: []string{"customer.cloud.mattermost.com"},
			dnsEndpoints:    []string{},
			setupID: func(zoneName string) (zoneID string, err error) {
				return "", nil
			},
			setupDNS: func(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error) {
				return &cf.DNSRecordResponse{}, nil
			},
			awsZoneNameList: func() []string {
				return []string{"cloud.mattermost.com"}
			},
			expected: errors.New("no DNS endpoints provided for Cloudflare creation request"),
		},
		{
			description:     "failure with empty string DNS endpoint",
			customerDNSName: []string{"customer.cloud.mattermost.com"},
			dnsEndpoints:    []string{""},
			setupID: func(zoneName string) (zoneID string, err error) {
				return "", nil
			},
			setupDNS: func(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error) {
				return &cf.DNSRecordResponse{}, nil
			},
			awsZoneNameList: func() []string {
				return []string{"cloud.mattermost.com"}
			},
			expected: errors.New("DNS endpoint was an empty string"),
		},
		{
			description:     "failure with zone ID fetching",
			customerDNSName: []string{"customer.cloud.mattermost.com"},
			dnsEndpoints:    []string{"load.balancer.endpoint"},
			setupID: func(zoneName string) (zoneID string, err error) {
				return "", errors.New("failed to get zone ID")
			},
			setupDNS: func(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error) {
				return &cf.DNSRecordResponse{}, nil
			},
			awsZoneNameList: func() []string {
				return []string{"cloud.mattermost.com"}
			},
			expected: errors.New("failed to fetch Zone ID from Cloudflare: failed to get zone ID"),
		},
	}

	for _, s := range samples {
		t.Run(s.description, func(t *testing.T) {
			mockCF.mockGetZoneID = s.setupID
			mockCF.mockCreateDNSRecord = s.setupDNS
			MockAWS.mockGetPublicHostedZoneNames = s.awsZoneNameList
			client := NewClientWithToken(mockCF, MockAWS)
			err := client.CreateDNSRecords(s.customerDNSName, s.dnsEndpoints, logger)
			if s.expected != nil {
				assert.EqualError(t, s.expected, err.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

//// TODO: adjust to multiple
func TestDeleteDNSRecord(t *testing.T) {
	logger := testlib.MakeLogger(t)

	mockCF := &MockCloudflare{}
	mockAWS := &MockAWSClient{}
	samples := []struct {
		description      string
		customerDNSNames []string
		zoneID           string
		setupZoneID          func(zoneName string) (zoneID string, err error)
		setupDeleteDNSRecord func(ctx context.Context, zoneID, recordID string) error
		setupDNSRecord       func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error)
		awsZoneNameList      func() []string
		expected             error
	}{
		{
			description:      "success path",
			customerDNSNames: []string{"customer.cloud.mattermost.com"},
			zoneID:           "RANDOMDIDFROMCLOUDFLARE",
			setupZoneID: func(zoneName string) (zoneID string, err error) {
				return "RANDOMDIDFROMCLOUDFLARE", nil
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
			awsZoneNameList: func() []string {
				return []string{"cloud.mattermost.com"}
			},
			expected: nil,
		},
		{
			description:      "success with 2 zone names",
			customerDNSNames: []string{"customer.cloud.mattermost.com"},
			zoneID:           "RANDOMDIDFROMCLOUDFLARE",
			setupZoneID: func(zoneName string) (zoneID string, err error) {
				return "RANDOMDIDFROMCLOUDFLARE", nil
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
			awsZoneNameList: func() []string {
				return []string{"cloud.mattermost.com", "cloud.test.mattermost.com"}
			},
			expected: nil,
		},
		{
			description:      "success with multiple domain names",
			customerDNSNames: []string{"customer.cloud.mattermost.com","customer.cloud.mattermost.io"},
			zoneID:           "RANDOMDIDFROMCLOUDFLARE",
			setupZoneID: func(zoneName string) (zoneID string, err error) {
				return "RANDOMDIDFROMCLOUDFLARE", nil
			},
			setupDeleteDNSRecord: func(ctx context.Context, zoneID, recordID string) error {
				return nil
			},
			setupDNSRecord: func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error) {
				return []cf.DNSRecord{
					{ID: "CLOUDFLARERECORDID"},
					{ID: "CLOUDFLARERECORDID2"},
				}, nil
			},
			awsZoneNameList: func() []string {
				return []string{"cloud.mattermost.com", "cloud.mattermost.io"}
			},
			expected: nil,
		},
		{
			description:      "failure to get zone ID",
			customerDNSNames: []string{"customer.cloud.mattermost.com"},
			zoneID:           "RANDOMDIDFROMCLOUDFLARE",
			setupZoneID: func(zoneName string) (zoneID string, err error) {
				return "", errors.New("failed to get zone ID")
			},
			setupDeleteDNSRecord: func(ctx context.Context, zoneID, recordID string) error {
				return nil
			},
			setupDNSRecord: func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error) {
				return []cf.DNSRecord{}, nil
			},
			awsZoneNameList: func() []string {
				return []string{"cloud.mattermost.com"}
			},
			expected: errors.New("failed to fetch Zone ID from Cloudflare: failed to get zone ID"),
		},
		{
			description:      "failure to get record ID",
			customerDNSNames: []string{"customer.cloud.mattermost.com"},
			zoneID:           "RANDOMDIDFROMCLOUDFLARE",
			setupZoneID: func(zoneName string) (zoneID string, err error) {
				return "RANDOMDIDFROMCLOUDFLARE", nil
			},
			setupDeleteDNSRecord: func(ctx context.Context, zoneID, recordID string) error {
				return nil
			},
			setupDNSRecord: func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error) {
				return []cf.DNSRecord{}, errors.New("failed to get DNS records")
			},
			awsZoneNameList: func() []string {
				return []string{"cloud.mattermost.com"}
			},
			expected: errors.New("Failed to get record ID from Cloudflare for DNS: customer.cloud.mattermost.com: failed to get DNS Record ID from Cloudflare: failed to get DNS records"),
		},
		{
			description:      "failure to delete DNS record",
			customerDNSNames: []string{"customer.cloud.mattermost.com"},
			zoneID:           "RANDOMDIDFROMCLOUDFLARE",
			setupZoneID: func(zoneName string) (zoneID string, err error) {
				return "RANDOMDIDFROMCLOUDFLARE", nil
			},
			setupDeleteDNSRecord: func(ctx context.Context, zoneID, recordID string) error {
				return errors.New("failed to delete DNS record")
			},
			setupDNSRecord: func(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error) {
				return []cf.DNSRecord{}, errors.New("failed to get DNS records")
			},
			awsZoneNameList: func() []string {
				return []string{"cloud.mattermost.com"}
			},
			expected: errors.New("Failed to get record ID from Cloudflare for DNS: customer.cloud.mattermost.com: failed to get DNS Record ID from Cloudflare: failed to get DNS records"),
		},
	}

	for _, s := range samples {
		t.Run(s.description, func(t *testing.T) {
			mockCF.mockGetZoneID = s.setupZoneID
			mockCF.mockDeleteDNSRecord = s.setupDeleteDNSRecord
			mockCF.mockDNSRecords = s.setupDNSRecord
			mockAWS.mockGetPublicHostedZoneNames = s.awsZoneNameList
			client := NewClientWithToken(mockCF, mockAWS)
			err := client.DeleteDNSRecords(s.customerDNSNames, logger)
			if s.expected != nil {
				assert.EqualError(t, s.expected, err.Error())
			}
		})
	}
}
