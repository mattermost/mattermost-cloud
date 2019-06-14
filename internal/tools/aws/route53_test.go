package aws

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

var testDNSName = "example.mattermost.com"

func (api *mockAPI) getRoute53Client() (*route53.Route53, error) {
	return nil, api.returnedError
}

func (api *mockAPI) changeResourceRecordSets(svc *route53.Route53, input *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error) {
	return nil, api.returnedError
}

func (api *mockAPI) listResourceRecordSets(svc *route53.Route53, input *route53.ListResourceRecordSetsInput) (*route53.ListResourceRecordSetsOutput, error) {
	return &route53.ListResourceRecordSetsOutput{
		IsTruncated: &api.returnedTruncated,
		ResourceRecordSets: []*route53.ResourceRecordSet{
			{
				Name: &testDNSName,
			},
		},
	}, api.returnedError
}

func TestCreateCNAME(t *testing.T) {
	tests := []struct {
		name        string
		dnsName     string
		endpoints   []string
		mockError   error
		expectError bool
	}{
		{
			"no endpoints",
			"dns1",
			[]string{},
			nil,
			true,
		}, {
			"one endpoints",
			"dns2",
			[]string{"example.mattermost.com"},
			nil,
			false,
		}, {
			"two endpoints",
			"dns3",
			[]string{"example1.mattermost.com", "example2.mattermost.com"},
			nil,
			false,
		}, {
			"session client error",
			"dns4",
			[]string{"example1.mattermost.com", "example2.mattermost.com"},
			errors.New("mock api error"),
			true,
		},
	}

	logger := logrus.New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Client{
				hostedZoneID: "ABCDEFGH",
				api:          &mockAPI{returnedError: tt.mockError},
			}

			err := a.CreateCNAME(tt.dnsName, tt.endpoints, logger)
			switch tt.expectError {
			case true:
				assert.Error(t, err)
			case false:
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeleteCNAME(t *testing.T) {
	tests := []struct {
		name          string
		dnsName       string
		mockError     error
		mockTruncated bool
		expectError   bool
	}{
		{
			"one endpoints, matching",
			testDNSName,
			nil,
			false,
			false,
		}, {
			"two endpoints, no matching",
			"no-matching",
			nil,
			false,
			false,
		}, {
			"session client error",
			"dns4",
			errors.New("mock api error"),
			false,
			true,
		},
	}

	logger := logrus.New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Client{
				hostedZoneID: "ABCDEFGH",
				api:          &mockAPI{returnedError: tt.mockError},
			}

			err := a.DeleteCNAME(tt.dnsName, logger)
			switch tt.expectError {
			case true:
				assert.Error(t, err)
			case false:
				assert.NoError(t, err)
			}
		})
	}
}
