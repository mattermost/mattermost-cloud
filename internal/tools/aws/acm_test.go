package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
)

func (a *AWSTestSuite) TestGetCertificateSummaryByTag() {
	gomock.InOrder(
		a.Mocks.API.ACM.EXPECT().
			ListCertificates(gomock.Any()).
			Return(&acm.ListCertificatesOutput{
				CertificateSummaryList: []*acm.CertificateSummary{
					{
						CertificateArn: aws.String(a.CertifcateARN + "a"),
					},
					{
						CertificateArn: aws.String(a.CertifcateARN + "b"),
					},
				},
				NextToken: aws.String("token"),
			}, nil).
			Times(1),

		a.Mocks.API.ACM.EXPECT().
			ListTagsForCertificate(gomock.Any()).
			Return(&acm.ListTagsForCertificateOutput{}, nil).
			Times(2),

		a.Mocks.API.ACM.EXPECT().
			ListCertificates(gomock.Any()).
			Return(&acm.ListCertificatesOutput{
				CertificateSummaryList: []*acm.CertificateSummary{
					{
						CertificateArn: aws.String(a.CertifcateARN),
					},
				},
			}, nil).
			Times(1),

		a.Mocks.API.ACM.EXPECT().
			ListTagsForCertificate(gomock.Any()).
			Return(&acm.ListTagsForCertificateOutput{
				Tags: []*acm.Tag{
					{
						Key:   aws.String("MattermostCloudInstallationCertificates"),
						Value: aws.String("value"),
					},
				},
			}, nil).
			Times(1),
	)

	summary, err := a.Mocks.AWS.GetCertificateSummaryByTag(DefaultInstallCertificatesTagKey, "value", a.Mocks.Log.Logger)
	a.Assert().NoError(err)
	a.Assert().NotNil(summary)
	a.Assert().Equal(a.CertifcateARN, *summary.CertificateArn)
}

func (a *AWSTestSuite) TestGetCertificateSummaryByTagNotFound() {
	gomock.InOrder(
		a.Mocks.API.ACM.EXPECT().
			ListCertificates(gomock.Any()).
			Return(&acm.ListCertificatesOutput{
				CertificateSummaryList: []*acm.CertificateSummary{
					{
						CertificateArn: aws.String(a.CertifcateARN + "a"),
					},
					{
						CertificateArn: aws.String(a.CertifcateARN + "b"),
					},
				},
			}, nil).
			Times(1),

		a.Mocks.API.ACM.EXPECT().
			ListTagsForCertificate(gomock.Any()).
			Return(&acm.ListTagsForCertificateOutput{
				Tags: []*acm.Tag{
					{
						Key:   aws.String("MattermostCloudInstallationCertificates"),
						Value: aws.String("value"),
					},
				},
			}, nil).
			Times(2),
	)

	summary, err := a.Mocks.AWS.GetCertificateSummaryByTag(DefaultInstallCertificatesTagKey, "not_found", a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("no certificate was found under tag:MattermostCloudInstallationCertificates:not_found", err.Error())
	a.Assert().Nil(summary)
}

func (a *AWSTestSuite) TestGetCertificateSummaryByTagCertListError() {
	gomock.InOrder(
		a.Mocks.API.ACM.EXPECT().
			ListCertificates(gomock.Any()).
			Return(nil, errors.New("list certificates error")).
			Times(1),

		a.Mocks.API.ACM.EXPECT().ListTagsForCertificate(gomock.Any()).Times(0),
	)

	summary, err := a.Mocks.AWS.GetCertificateSummaryByTag(DefaultInstallCertificatesTagKey, "not_found", a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("error fetching certificates: list certificates error", err.Error())
	a.Assert().Nil(summary)
}

func (a *AWSTestSuite) TestGetCertificateSummaryByTagListTagsError() {
	gomock.InOrder(
		a.Mocks.API.ACM.EXPECT().
			ListCertificates(gomock.Any()).
			Return(&acm.ListCertificatesOutput{
				CertificateSummaryList: []*acm.CertificateSummary{
					{
						CertificateArn: aws.String(a.CertifcateARN + "a"),
					},
					{
						CertificateArn: aws.String(a.CertifcateARN + "b"),
					},
				},
			}, nil).
			Times(1),

		a.Mocks.API.ACM.EXPECT().
			ListTagsForCertificate(gomock.Any()).
			Return(nil, errors.New("list tags error")).
			Times(1),
	)

	summary, err := a.Mocks.AWS.GetCertificateSummaryByTag(DefaultInstallCertificatesTagKey, "not_found", a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("error listing tags for certificate arn:aws:certificate::123456789012a: list tags error", err.Error())
	a.Assert().Nil(summary)
}
