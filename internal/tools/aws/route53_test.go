package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/golang/mock/gomock"
	testlib "github.com/mattermost/mattermost-cloud/internal/testlib"
	log "github.com/sirupsen/logrus"
)

// var next *string
// 	for {
// 		zoneList, err := a.Service(logger).route53.ListHostedZones(&route53.ListHostedZonesInput{Marker: next})
// 		if err != nil {
// 			return "", errors.Wrapf(err, "listing hosted all zones")
// 		}

// 		for _, zone := range zoneList.HostedZones {
// 			id, err := parseHostedZoneResourceID(zone)
// 			if err != nil {
// 				return "", errors.Wrapf(err, "when parsing hosted zone: %s", zone.String())
// 			}

// 			tagList, err := a.Service(logger).route53.ListTagsForResource(&route53.ListTagsForResourceInput{
// 				ResourceId:   aws.String(id),
// 				ResourceType: aws.String(hostedZoneResourceType),
// 			})
// 			if err != nil {
// 				return "", err
// 			}

// 			for _, resourceTag := range tagList.ResourceTagSet.Tags {
// 				if tag.Compare(resourceTag) {
// 					return id, nil
// 				}
// 			}
// 		}

// 		if zoneList.Marker == nil || *zoneList.Marker == "" {
// 			break
// 		}
// 		next = zoneList.Marker
// 	}

// 	return "", errors.Errorf("no hosted zone ID associated with tag: %s", tag.String())

func (a *AWSTestSuite) TestRoute53CreatePublicCNAME() {
	gomock.InOrder(
		a.Mocks.API.Route53.EXPECT().
			ListHostedZones(&route53.ListHostedZonesInput{}).
			Return(&route53.ListHostedZonesOutput{
				HostedZones: []*route53.HostedZone{
					&route53.HostedZone{
						Id: aws.String("ZWI3O6O6N782A"),
					},
				},
				Marker:      aws.String("next"),
				IsTruncated: aws.Bool(true),
			}, nil).
			Times(1),

		a.Mocks.API.Route53.EXPECT().
			ListTagsForResource(gomock.Any()).
			Do(func(input *route53.ListTagsForResourceInput) {
				a.Assert().Equal("ZWI3O6O6N782A", *input.ResourceId)
				a.Assert().Equal("hostedzone", *input.ResourceType)
			}).
			Return(&route53.ListTagsForResourceOutput{
				ResourceTagSet: &route53.ResourceTagSet{
					Tags: []*route53.Tag{
						&route53.Tag{
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
					&route53.HostedZone{
						Id: aws.String("ZWI3O6O6N782C"),
					},
				},
				IsTruncated: aws.Bool(false),
			}, nil).
			Times(1),

		a.Mocks.API.Route53.EXPECT().
			ListTagsForResource(gomock.Any()).
			Do(func(input *route53.ListTagsForResourceInput) {
				a.Assert().Equal("ZWI3O6O6N782C", *input.ResourceId)
				a.Assert().Equal("hostedzone", *input.ResourceType)
			}).
			Return(&route53.ListTagsForResourceOutput{
				ResourceTagSet: &route53.ResourceTagSet{
					Tags: []*route53.Tag{
						&route53.Tag{
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
				a.Assert().Equal("ZWI3O6O6N782C", *input.HostedZoneId)
			}).
			Return(&route53.ChangeResourceRecordSetsOutput{}, nil),

		a.Mocks.LOG.Logger.EXPECT().WithFields(log.Fields{
			"route53-dns-value":      "mattermost.com",
			"route53-dns-endpoints":  []string{"example.mattermost.com"},
			"route53-hosted-zone-id": "ZWI3O6O6N782C",
		}).
			Return(testlib.NewLoggerEntry()).
			Times(1),
	)

	err := a.Mocks.AWS.CreatePublicCNAME("mattermost.com", []string{"example.mattermost.com"}, a.Mocks.LOG.Logger)
	a.Assert().NoError(err)
}

func (a *AWSTestSuite) TestRoute53DeletePublicCNAME() {
	a.T().Skip()
}
