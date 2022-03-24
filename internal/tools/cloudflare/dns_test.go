// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package cloudflare

import (
	"context"
	"errors"
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/sirupsen/logrus"

	cf "github.com/cloudflare/cloudflare-go"
	"github.com/stretchr/testify/assert"
)

type mockCloudflare struct {
	mockGetZoneId       func(zoneName string) (zoneID string, err error)
	mockGetZoneName     func(zoneNameList []string, customerDnsName string) (zoneName string, found bool)
	mockGetRecordId     func(zoneID, customerDnsName string, logger logrus.FieldLogger) (recordID string, err error)
	mockCreateDNSRecord func(customerDnsName string, zoneNameList []string, dnsEndpoints []string, logger logrus.FieldLogger) error
	mockDeleteDNSRecord func(customerDnsName string, zoneNameList []string, logger logrus.FieldLogger) error
}

func (e *mockCloudflare) ZoneIDByName(zoneName string) (string, error) {
	return e.mockGetZoneId(zoneName)
}

func (e *mockCloudflare) getZoneName(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
	return e.mockGetZoneName(zoneNameList, customerDnsName)
}

func (e *mockCloudflare) getRecordId(zoneID, customerDnsName string, logger logrus.FieldLogger) (recordID string, err error) {
	return e.mockGetRecordId(zoneID, customerDnsName, logger)
}

func (e *mockCloudflare) DNSRecords(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error) {
	return nil, nil
}
func (e *mockCloudflare) CreateDNSRecord(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error) {
	return nil, nil
}

func (e *mockCloudflare) DeleteDNSRecord(ctx context.Context, zoneID, recordID string) error {
	return nil
}

func TestGetZoneId(t *testing.T) {
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
		mockCF.mockGetZoneId = s.setup
		client := NewClientWithToken(mockCF)
		id, _ := client.getZoneId(s.zoneName)
		assert.Equal(t, s.expected, id)
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
		customerDnsName string
		setup           func(zoneNameList []string, customerDnsName string) (zoneName string, found bool)
		expected        Expected
	}{
		{
			description:     "success with 1 zone name in the list",
			zoneNameList:    []string{"cloud.mattermost.com"},
			customerDnsName: "customer.cloud.mattermost.com",
			setup: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "cloud.mattermost.com", true
			},
			expected: Expected{"cloud.mattermost.com", true},
		},
		{
			description:     "success with 2 zone name in the list",
			zoneNameList:    []string{"cloud.mattermost.com", "cloud.test.mattermost.com"},
			customerDnsName: "customer.cloud.mattermost.com",
			setup: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "cloud.mattermost.com", true
			},
			expected: Expected{"cloud.mattermost.com", true},
		},
		{
			description:     "failure with 1 zone name in the list",
			zoneNameList:    []string{"cloud.env.mattermost.com"},
			customerDnsName: "customer.cloud.mattermost.com",
			setup: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "", false
			},
			expected: Expected{"", false},
		},
		{
			description:     "failure with 2 zone name in the list",
			zoneNameList:    []string{"cloud.env.mattermost.com", "cloud.test.mattermost.com"},
			customerDnsName: "customer.cloud.mattermost.com",
			setup: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "", false
			},
			expected: Expected{"", false},
		},
		{
			description:     "failure empty zone name in the list",
			zoneNameList:    []string{},
			customerDnsName: "customer.cloud.mattermost.com",
			setup: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "", false
			},
			expected: Expected{"", false},
		},
		{
			description:     "failure empty customer DNS name",
			zoneNameList:    []string{"cloud.env.mattermost.com", "cloud.test.mattermost.com"},
			customerDnsName: "",
			setup: func(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
				return "", false
			},
			expected: Expected{"", false},
		},
	}

	for _, s := range samples {
		mockCF.mockGetZoneName = s.setup
		client := NewClientWithToken(mockCF)
		name, found := client.getZoneName(s.zoneNameList, s.customerDnsName)
		result := Expected{name, found}
		assert.Equal(t, s.expected, result, s.description)
	}

}

func TestGetRecordId(t *testing.T) {
	type Expected struct {
		string
		error
	}
	mockCF := &mockCloudflare{}
	samples := []struct {
		description     string
		zoneID          string
		customerDnsName string
		logger          *testlib.MockedFieldLogger
		setup           func(zoneID, customerDnsName string, logger logrus.FieldLogger) (recordID string, err error)
		expected        Expected
	}{
		{
			description:     "success with 1 zone name in the list",
			zoneID:          "THISISAZONEIDFROMCLOUDFLARE",
			customerDnsName: "customer.cloud.mattermost.com",
			//logger:          logrus.Info("failed creating third party resources"),
			//logger:          logger.Info("failed creating third party resources"),
			setup: func(zoneID, customerDnsName string, logger logrus.FieldLogger) (recordID string, err error) {
				return "CLOUDFLARERECORDID", nil
			},
			expected: Expected{"CLOUDFLARERECORDID", nil},
		},
	}

	for _, s := range samples {
		mockCF.mockGetRecordId = s.setup
		client := NewClientWithToken(mockCF)
		name, found := client.getRecordId(s.zoneID, s.customerDnsName, s.logger)
		result := Expected{name, found}
		println(s.description)
		assert.Equal(t, s.expected, result, s.description)
	}

}
