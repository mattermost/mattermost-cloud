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
			expected:        Expected{"cloud.mattermost.com", true},
		},
		{
			description:     "success with 2 zone name in the list",
			zoneNameList:    []string{"cloud.mattermost.com", "cloud.test.mattermost.com"},
			customerDNSName: "customer.cloud.mattermost.com",
			expected:        Expected{"cloud.mattermost.com", true},
		},
		{
			description:     "failure with 1 zone name in the list",
			zoneNameList:    []string{"cloud.env.mattermost.com"},
			customerDNSName: "customer.cloud.mattermost.com",
			expected:        Expected{"", false},
		},
		{
			description:     "failure with 2 zone name in the list",
			zoneNameList:    []string{"cloud.env.mattermost.com", "cloud.test.mattermost.com"},
			customerDNSName: "customer.cloud.mattermost.com",
			expected:        Expected{"", false},
		},
		{
			description:     "failure empty zone name in the list",
			zoneNameList:    []string{},
			customerDNSName: "customer.cloud.mattermost.com",
			expected:        Expected{"", false},
		},
		{
			description:     "failure empty customer DNS name",
			zoneNameList:    []string{"cloud.env.mattermost.com", "cloud.test.mattermost.com"},
			customerDNSName: "",
			expected:        Expected{"", false},
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
		description         string
		zoneID              string
		customerDNSName     string
		setupListDNSRecords func(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error)
		expected            Expected
	}{
		{
			description:     "success with 1 zone name in the list",
			zoneID:          "THISISAZONEIDFROMCLOUDFLARE",
			customerDNSName: "customer.cloud.mattermost.com",
			setupListDNSRecords: func(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error) {
				return []cf.DNSRecord{
					{
						ID: "CLOUDFLARERECORDID",
					},
				}, nil, nil
			},
			expected: Expected{[]string{"CLOUDFLARERECORDID"}, nil},
		},
		{
			description:     "success with multiple ids",
			zoneID:          "THISISAZONEIDFROMCLOUDFLARE",
			customerDNSName: "customer.cloud.mattermost.com",
			setupListDNSRecords: func(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error) {
				return []cf.DNSRecord{
					{ID: "CLOUDFLARERECORDID"},
					{ID: "CLOUDFLARERECORDID2"},
				}, nil, nil
			},
			expected: Expected{[]string{"CLOUDFLARERECORDID", "CLOUDFLARERECORDID2"}, nil},
		},
		{
			description:     "non existing zone ID at Cloudflare",
			zoneID:          "NONEXISTINGIDATCLOUDFLARE",
			customerDNSName: "customer.cloud.mattermost.com",
			setupListDNSRecords: func(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error) {
				return []cf.DNSRecord{}, nil, nil
			},
			expected: Expected{nil, nil},
		},
		{
			description:     "error while calling cloudflare API",
			zoneID:          "NONEXISTINGIDATCLOUDFLARE",
			customerDNSName: "customer.cloud.mattermost.com",
			setupListDNSRecords: func(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error) {
				return []cf.DNSRecord{}, nil, errors.New("Cloudflare API error")
			},
			expected: Expected{nil, errors.New("failed to get DNS Record ID from Cloudflare: Cloudflare API error")},
		},
	}

	for _, s := range samples {
		t.Run(s.description, func(t *testing.T) {
			mockCF.mockListDNSRecords = s.setupListDNSRecords
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
		createDNS       func(context.Context, *cf.ResourceContainer, cf.CreateDNSRecordParams) (cf.DNSRecord, error)
		updateDNS       func(ctx context.Context, rc *cf.ResourceContainer, params cf.UpdateDNSRecordParams) (cf.DNSRecord, error)
		getDNSRecords   func(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error)
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
			createDNS: func(ctx context.Context, rc *cf.ResourceContainer, params cf.CreateDNSRecordParams) (cf.DNSRecord, error) {
				return cf.DNSRecord{
					ID: "CLOUDFLARERECORDID",
				}, nil
			},
			getDNSRecords: func(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error) {
				return []cf.DNSRecord{}, nil, nil
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
			createDNS: func(ctx context.Context, rc *cf.ResourceContainer, params cf.CreateDNSRecordParams) (cf.DNSRecord, error) {
				return cf.DNSRecord{
					ID: "CLOUDFLARERECORDID",
				}, nil
			},
			getDNSRecords: func(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error) {
				return []cf.DNSRecord{}, nil, nil
			},
			awsZoneNameList: func() []string {
				return []string{"cloud.mattermost.com", "cloud.test.mattermost.com"}
			},
			expected: nil,
		},
		{
			description:     "success when record already exists",
			customerDNSName: []string{"customer.cloud.mattermost.com"},
			dnsEndpoints:    []string{"load.balancer.endpoint"},
			setupID: func(zoneName string) (zoneID string, err error) {
				return "CLOUDFLAREZONEID", nil
			},
			getDNSRecords: func(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error) {
				trueVal := true
				return []cf.DNSRecord{
					{
						Name:    params.Name,
						ID:      "CLOUDFLARERECORDID",
						ZoneID:  rc.Identifier,
						Content: "load.balancer.endpoint",
						TTL:     1,
						Proxied: &trueVal,
					},
				}, nil, nil
			},
			awsZoneNameList: func() []string {
				return []string{"cloud.mattermost.com"}
			},
			expected: nil,
		},
		{
			description:     "success when record already exists",
			customerDNSName: []string{"customer.cloud.mattermost.com"},
			dnsEndpoints:    []string{"load.balancer.endpoint"},
			setupID: func(zoneName string) (zoneID string, err error) {
				return "CLOUDFLAREZONEID", nil
			},
			getDNSRecords: func(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error) {
				trueVal := true
				return []cf.DNSRecord{
					{
						Name:    params.Name,
						ID:      "CLOUDFLARERECORDID",
						ZoneID:  rc.Identifier,
						Content: "load.balancer.endpoint",
						TTL:     2,
						Proxied: &trueVal,
					},
				}, nil, nil
			},
			updateDNS: func(ctx context.Context, rc *cf.ResourceContainer, params cf.UpdateDNSRecordParams) (cf.DNSRecord, error) {
				assert.Equal(t, "CLOUDFLARERECORDID", params.ID)
				assert.Equal(t, "CLOUDFLAREZONEID", rc.Identifier)
				return cf.DNSRecord{}, nil
			},
			awsZoneNameList: func() []string {
				return []string{"cloud.mattermost.com"}
			},
			expected: nil,
		},

		{
			description:     "success with multiple domain names",
			customerDNSName: []string{"customer.cloud.mattermost.com", "customer.cloud.mattermost.io"},
			dnsEndpoints:    []string{"load.balancer.endpoint"},
			setupID: func(zoneName string) (zoneID string, err error) {
				return "RANDOMDIDFROMCLOUDFLARE", nil
			},
			createDNS: func(ctx context.Context, rc *cf.ResourceContainer, params cf.CreateDNSRecordParams) (cf.DNSRecord, error) {
				return cf.DNSRecord{
					ID: "CLOUDFLARERECORDID",
				}, nil
			},
			getDNSRecords: func(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error) {
				return []cf.DNSRecord{}, nil, nil
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
			expected:        errors.New("creating record for more than one endpoint not supported"),
		},
		{
			description:     "failure with no domain names",
			customerDNSName: nil,
			dnsEndpoints:    []string{"load.balancer.endpoint"},
			setupID:         nil,
			createDNS:       nil,
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
			createDNS: func(ctx context.Context, rc *cf.ResourceContainer, params cf.CreateDNSRecordParams) (cf.DNSRecord, error) {
				return cf.DNSRecord{}, nil
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
			createDNS: func(ctx context.Context, rc *cf.ResourceContainer, params cf.CreateDNSRecordParams) (cf.DNSRecord, error) {
				return cf.DNSRecord{}, nil
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
			createDNS: func(ctx context.Context, rc *cf.ResourceContainer, params cf.CreateDNSRecordParams) (cf.DNSRecord, error) {
				return cf.DNSRecord{}, nil
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
			createDNS: func(ctx context.Context, rc *cf.ResourceContainer, params cf.CreateDNSRecordParams) (cf.DNSRecord, error) {
				return cf.DNSRecord{}, nil
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
			mockCF.mockCreateDNSRecord = s.createDNS
			mockCF.mockListDNSRecords = s.getDNSRecords
			mockCF.mockUpdateDNSRecord = s.updateDNS
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

func TestDeleteDNSRecord(t *testing.T) {
	logger := testlib.MakeLogger(t)

	mockCF := &MockCloudflare{}
	mockAWS := &MockAWSClient{}
	samples := []struct {
		description          string
		customerDNSNames     []string
		zoneID               string
		setupZoneID          func(zoneName string) (zoneID string, err error)
		setupDeleteDNSRecord func(ctx context.Context, rc *cf.ResourceContainer, recordID string) error
		setupListDNSRecord   func(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error)
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
			setupDeleteDNSRecord: func(ctx context.Context, rc *cf.ResourceContainer, recordID string) error {
				return nil
			},
			setupListDNSRecord: func(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error) {
				return []cf.DNSRecord{
					{
						ID: "CLOUDFLARERECORDID",
					},
				}, nil, nil
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
			setupDeleteDNSRecord: func(ctx context.Context, rc *cf.ResourceContainer, recordID string) error {
				return nil
			},
			setupListDNSRecord: func(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error) {
				return []cf.DNSRecord{
					{
						ID: "CLOUDFLARERECORDID",
					},
				}, nil, nil
			},
			awsZoneNameList: func() []string {
				return []string{"cloud.mattermost.com", "cloud.test.mattermost.com"}
			},
			expected: nil,
		},
		{
			description:      "success with multiple domain names",
			customerDNSNames: []string{"customer.cloud.mattermost.com", "customer.cloud.mattermost.io"},
			zoneID:           "RANDOMDIDFROMCLOUDFLARE",
			setupZoneID: func(zoneName string) (zoneID string, err error) {
				return "RANDOMDIDFROMCLOUDFLARE", nil
			},
			setupDeleteDNSRecord: func(ctx context.Context, rc *cf.ResourceContainer, recordID string) error {
				return nil
			},
			setupListDNSRecord: func(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error) {
				return []cf.DNSRecord{
					{ID: "CLOUDFLARERECORDID"},
					{ID: "CLOUDFLARERECORDID2"},
				}, nil, nil
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
			setupDeleteDNSRecord: func(ctx context.Context, rc *cf.ResourceContainer, recordID string) error {
				return nil
			},
			setupListDNSRecord: func(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error) {
				return []cf.DNSRecord{}, nil, nil
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
			setupDeleteDNSRecord: func(ctx context.Context, rc *cf.ResourceContainer, recordID string) error {
				return nil
			},
			setupListDNSRecord: func(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error) {
				return []cf.DNSRecord{}, nil, errors.New("failed to get DNS records")
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
			setupDeleteDNSRecord: func(ctx context.Context, rc *cf.ResourceContainer, recordID string) error {
				return errors.New("failed to delete DNS record")
			},
			setupListDNSRecord: func(ctx context.Context, rc *cf.ResourceContainer, params cf.ListDNSRecordsParams) ([]cf.DNSRecord, *cf.ResultInfo, error) {
				return []cf.DNSRecord{}, nil, errors.New("failed to get DNS records")
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
			mockCF.mockListDNSRecords = s.setupListDNSRecord
			mockAWS.mockGetPublicHostedZoneNames = s.awsZoneNameList
			client := NewClientWithToken(mockCF, mockAWS)
			err := client.DeleteDNSRecords(s.customerDNSNames, logger)
			if s.expected != nil {
				assert.EqualError(t, s.expected, err.Error())
			}
		})
	}
}
