// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/golang/mock/gomock"
	testlib "github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (a *AWSTestSuite) TestRoute53CreatePublicCNAME() {
	gomock.InOrder(
		a.Mocks.API.Route53.EXPECT().
			ListHostedZones(&route53.ListHostedZonesInput{}).
			Return(&route53.ListHostedZonesOutput{
				HostedZones: []*route53.HostedZone{
					{
						Id: aws.String(a.HostedZoneID),
					},
				},
				Marker:      aws.String("next"),
				IsTruncated: aws.Bool(true),
			}, nil).
			Times(1),

		a.Mocks.API.Route53.EXPECT().
			ListTagsForResource(gomock.Any()).
			Do(func(input *route53.ListTagsForResourceInput) {
				a.Assert().Equal(a.HostedZoneID, *input.ResourceId)
				a.Assert().Equal("hostedzone", *input.ResourceType)
			}).
			Return(&route53.ListTagsForResourceOutput{
				ResourceTagSet: &route53.ResourceTagSet{
					Tags: []*route53.Tag{
						{
							Key:   aws.String("random-key"),
							Value: aws.String("random-value"),
						},
					},
				},
			}, nil).
			Times(1),

		a.Mocks.API.Route53.EXPECT().
			ListHostedZones(gomock.Any()).
			Return(&route53.ListHostedZonesOutput{
				HostedZones: []*route53.HostedZone{
					{
						Id: aws.String(a.HostedZoneID),
					},
				},
				IsTruncated: aws.Bool(false),
			}, nil).
			Times(1),

		a.Mocks.API.Route53.EXPECT().
			ListTagsForResource(gomock.Any()).
			Do(func(input *route53.ListTagsForResourceInput) {
				a.Assert().Equal(a.HostedZoneID, *input.ResourceId)
				a.Assert().Equal("hostedzone", *input.ResourceType)
			}).
			Return(&route53.ListTagsForResourceOutput{
				ResourceTagSet: &route53.ResourceTagSet{
					Tags: []*route53.Tag{
						{
							Key:   aws.String("MattermostCloudDNS"),
							Value: aws.String("public"),
						},
					},
				},
			}, nil).
			Times(1),

		a.Mocks.API.Route53.EXPECT().
			ChangeResourceRecordSets(gomock.Any()).
			Do(func(input *route53.ChangeResourceRecordSetsInput) {
				a.Assert().Equal("mattermost.com", *input.ChangeBatch.Changes[0].ResourceRecordSet.Name)
				a.Assert().Equal("example.mattermost.com", *input.ChangeBatch.Changes[0].ResourceRecordSet.ResourceRecords[0].Value)
				a.Assert().Equal(a.HostedZoneID, *input.HostedZoneId)
			}).
			Return(&route53.ChangeResourceRecordSetsOutput{}, nil),

		a.Mocks.Log.Logger.EXPECT().WithFields(log.Fields{
			"route53-dns-value":      "mattermost.com",
			"route53-dns-endpoints":  []string{"example.mattermost.com"},
			"route53-hosted-zone-id": a.HostedZoneID,
		}).
			Return(testlib.NewLoggerEntry()).
			Times(1),
	)

	err := a.Mocks.AWS.CreatePublicCNAME("mattermost.com", []string{"example.mattermost.com"}, a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}

func (a *AWSTestSuite) TestRoute53CreatePublicCNAMEListZonesError() {
	a.Mocks.API.Route53.EXPECT().
		ListHostedZones(&route53.ListHostedZonesInput{}).
		Return(nil, errors.New("invalid input")).
		Times(1)

	a.Mocks.API.Route53.EXPECT().ListTagsForResource(gomock.Any()).Times(0)
	a.Mocks.API.Route53.EXPECT().ChangeResourceRecordSets(gomock.Any()).Times(0)
	a.Mocks.Log.Logger.EXPECT().WithFields(gomock.Any()).Times(0)

	err := a.Mocks.AWS.CreatePublicCNAME("mattermost.com", []string{"example.mattermost.com"}, a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("unable to create a public CNAME: mattermost.com: listing hosted all zones: invalid input", err.Error())
}

func (a *AWSTestSuite) TestRoute53CreatePublicCNAMEListTagsError() {
	gomock.InOrder(
		a.Mocks.API.Route53.EXPECT().
			ListHostedZones(&route53.ListHostedZonesInput{}).
			Return(&route53.ListHostedZonesOutput{
				HostedZones: []*route53.HostedZone{
					{
						Id: aws.String(a.HostedZoneID),
					},
				},
				Marker:      aws.String("next"),
				IsTruncated: aws.Bool(true),
			}, nil).
			Times(1),

		a.Mocks.API.Route53.EXPECT().
			ListTagsForResource(gomock.Any()).
			Do(func(input *route53.ListTagsForResourceInput) {
				a.Assert().Equal(a.HostedZoneID, *input.ResourceId)
				a.Assert().Equal("hostedzone", *input.ResourceType)
			}).
			Return(nil, errors.New("region is not set")).
			Times(1),
	)

	a.Mocks.API.Route53.EXPECT().ChangeResourceRecordSets(gomock.Any()).Times(0)
	a.Mocks.Log.Logger.EXPECT().WithFields(gomock.Any()).Times(0)

	err := a.Mocks.AWS.CreatePublicCNAME("mattermost.com", []string{"example.mattermost.com"}, a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("unable to create a public CNAME: mattermost.com: region is not set", err.Error())
}

func (a *AWSTestSuite) TestRoute53CreatePublicCNAMEChangeRecordSetsError() {
	gomock.InOrder(
		a.Mocks.API.Route53.EXPECT().
			ListHostedZones(gomock.Any()).
			Return(&route53.ListHostedZonesOutput{
				HostedZones: []*route53.HostedZone{
					{
						Id: aws.String(a.HostedZoneID),
					},
				},
				IsTruncated: aws.Bool(false),
			}, nil).
			Times(1),

		a.Mocks.API.Route53.EXPECT().
			ListTagsForResource(gomock.Any()).
			Do(func(input *route53.ListTagsForResourceInput) {
				a.Assert().Equal(a.HostedZoneID, *input.ResourceId)
				a.Assert().Equal("hostedzone", *input.ResourceType)
			}).
			Return(&route53.ListTagsForResourceOutput{
				ResourceTagSet: &route53.ResourceTagSet{
					Tags: []*route53.Tag{
						{
							Key:   aws.String("MattermostCloudDNS"),
							Value: aws.String("public"),
						},
					},
				},
			}, nil).
			Times(1),

		a.Mocks.API.Route53.EXPECT().
			ChangeResourceRecordSets(gomock.Any()).
			Do(func(input *route53.ChangeResourceRecordSetsInput) {
				a.Assert().Equal("mattermost.com", *input.ChangeBatch.Changes[0].ResourceRecordSet.Name)
				a.Assert().Equal("example.mattermost.com", *input.ChangeBatch.Changes[0].ResourceRecordSet.ResourceRecords[0].Value)
				a.Assert().Equal(a.HostedZoneID, *input.HostedZoneId)
			}).
			Return(nil, errors.New("unable to change recordsets")).
			Times(1),
	)

	a.Mocks.Log.Logger.EXPECT().WithFields(gomock.Any()).Times(0)

	err := a.Mocks.AWS.CreatePublicCNAME("mattermost.com", []string{"example.mattermost.com"}, a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("unable to change recordsets", err.Error())
}

func (a *AWSTestSuite) TestRoute53CreatePublicCNAMENoHostedZone() {
	gomock.InOrder(
		a.Mocks.API.Route53.EXPECT().
			ListHostedZones(gomock.Any()).
			Return(&route53.ListHostedZonesOutput{
				HostedZones: []*route53.HostedZone{
					{
						Id: aws.String(a.HostedZoneID),
					},
				},
				IsTruncated: aws.Bool(false),
			}, nil).
			Times(1),

		a.Mocks.API.Route53.EXPECT().
			ListTagsForResource(gomock.Any()).
			Do(func(input *route53.ListTagsForResourceInput) {
				a.Assert().Equal(a.HostedZoneID, *input.ResourceId)
				a.Assert().Equal("hostedzone", *input.ResourceType)
			}).
			Return(&route53.ListTagsForResourceOutput{
				ResourceTagSet: &route53.ResourceTagSet{
					Tags: []*route53.Tag{
						{
							Key:   aws.String("random-key"),
							Value: aws.String("random-value"),
						},
					},
				},
			}, nil).
			Times(1),
	)

	err := a.Mocks.AWS.CreatePublicCNAME("mattermost.com", []string{"example.mattermost.com"}, a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("unable to create a public CNAME: mattermost.com: no hosted zone ID associated with tag: tag:MattermostCloudDNS:public", err.Error())
}

func (a *AWSTestSuite) TestRoute53DeletePublicCNAME() {
	a.T().Skip()
}
