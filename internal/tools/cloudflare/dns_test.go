// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package cloudflare

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	cf "github.com/cloudflare/cloudflare-go"
)

type mockCloudflare struct {
	cfClient *cf.API
	//mockNewClientWithToken func(token string) (*Client, error)
	mockGetZoneId func(zoneName string) (zoneID string, err error)
}

func (e *mockCloudflare) NewClientWithToken(token string) *Client {
	return e.mockNewClientWithToken(token)
}

func (e *mockCloudflare) getZoneId(zoneName string) (zoneID string, err error) {
	return e.mockGetZoneId(zoneName)
}

func TestGetZoneId(t *testing.T) {

	mockCF := mockCloudflare{}
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
