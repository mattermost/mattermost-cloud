package aws

import (
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/stretchr/testify/mock"
)

// TODO(gsagula): add tests for Route53.

func (a *AWSTestSuite) TestRoute53CreatePublicCNAME() {
	a.T().Skip()

	a.Mocks.API.Route53.On("ListHostedZones", mock.AnythingOfType("*route53.ListHostedZonesInput")).Return(&route53.ListHostedZonesOutput{
		HostedZones: []*route53.HostedZone{&route53.HostedZone{}},
	})

	err := a.Mocks.AWS.CreatePublicCNAME(a.DNSNameA, a.EndpointsA, a.Mocks.LOG.Logger)

	a.Assert().NoError(err)
	a.Mocks.API.Route53.AssertExpectations(a.T())
}

func (a *AWSTestSuite) TestRoute53DeletePublicCNAME() {
	a.T().Skip()

	a.Mocks.API.Route53.On("ListHostedZones", mock.AnythingOfType("*route53.ListHostedZonesInput")).Return(&route53.ListHostedZonesOutput{
		HostedZones: []*route53.HostedZone{&route53.HostedZone{}},
	})

	err := a.Mocks.AWS.DeletePublicCNAME(a.DNSNameA, a.Mocks.LOG.Logger)

	a.Assert().NoError(err)
	a.Mocks.API.Route53.AssertExpectations(a.T())
}
