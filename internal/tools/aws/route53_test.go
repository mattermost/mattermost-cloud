// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/route53"
	"github.com/golang/mock/gomock"
	testlib "github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (a *AWSTestSuite) TestRoute53CreatePublicCNAME() {
	gomock.InOrder(
		a.Mocks.API.Route53.EXPECT().
			ChangeResourceRecordSets(gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, input *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) {
				a.Assert().Equal("mattermost.com", *input.ChangeBatch.Changes[0].ResourceRecordSet.Name)
				a.Assert().Equal("example.mattermost.com", *input.ChangeBatch.Changes[0].ResourceRecordSet.ResourceRecords[0].Value)
			}).
			Return(&route53.ChangeResourceRecordSetsOutput{}, nil),

		a.Mocks.Log.Logger.EXPECT().WithFields(log.Fields{
			"route53-dns-value":      "mattermost.com",
			"route53-dns-endpoints":  []string{"example.mattermost.com"},
			"route53-hosted-zone-id": "HZONE2",
		}).
			Return(testlib.NewLoggerEntry()).
			Times(1),
	)

	err := a.Mocks.AWS.CreatePublicCNAME("mattermost.com", []string{"example.mattermost.com"}, "", a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}

func (a *AWSTestSuite) TestRoute53CreatePrivateCNAME() {
	gomock.InOrder(
		a.Mocks.API.Route53.EXPECT().
			ChangeResourceRecordSets(gomock.Any(), gomock.Any()).
			Do(func(context context.Context, input *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) {
				a.Assert().Equal("mattermost.com", *input.ChangeBatch.Changes[0].ResourceRecordSet.Name)
				a.Assert().Equal("example.mattermost.com", *input.ChangeBatch.Changes[0].ResourceRecordSet.ResourceRecords[0].Value)
				a.Assert().Equal(a.Mocks.AWS.GetPrivateHostedZoneID(), *input.HostedZoneId)
			}).
			Return(&route53.ChangeResourceRecordSetsOutput{}, nil),

		a.Mocks.Log.Logger.EXPECT().WithFields(log.Fields{
			"route53-dns-value":      "mattermost.com",
			"route53-dns-endpoints":  []string{"example.mattermost.com"},
			"route53-hosted-zone-id": a.Mocks.AWS.GetPrivateHostedZoneID(),
		}).
			Return(testlib.NewLoggerEntry()).
			Times(1),
	)

	err := a.Mocks.AWS.CreatePrivateCNAME("mattermost.com", []string{"example.mattermost.com"}, a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}

func (a *AWSTestSuite) TestRoute53CreatePublicCNAMEChangeRecordSetsError() {
	gomock.InOrder(
		a.Mocks.API.Route53.EXPECT().
			ChangeResourceRecordSets(gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, input *route53.ChangeResourceRecordSetsInput, optFns ...func(*route53.Options)) {
				a.Assert().Equal("mattermost.com", *input.ChangeBatch.Changes[0].ResourceRecordSet.Name)
				a.Assert().Equal("example.mattermost.com", *input.ChangeBatch.Changes[0].ResourceRecordSet.ResourceRecords[0].Value)
			}).
			Return(nil, errors.New("unable to change recordsets")).
			Times(1),
	)

	a.Mocks.Log.Logger.EXPECT().WithFields(gomock.Any()).Times(0)

	err := a.Mocks.AWS.CreatePublicCNAME("mattermost.com", []string{"example.mattermost.com"}, "", a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("unable to change recordsets", err.Error())
}

func (a *AWSTestSuite) TestRoute53DeletePublicCNAME() {
	a.T().Skip()
}
