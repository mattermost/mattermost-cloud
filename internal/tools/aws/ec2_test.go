package aws

import (
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func (a *AWSTestSuite) TestTagResource() {
	a.Mocks.API.EC2.On("CreateTags", mock.AnythingOfType("*ec2.CreateTagsInput")).Return(&ec2.CreateTagsOutput{}, nil)
	a.Mocks.LOG.WithFields(logrus.Fields{"tag-key": "tag-key", "tag-value": "tag-value"}).Once()

	err := a.Mocks.AWS.TagResource(a.ResourceID, "tag-key", "tag-value", a.Mocks.LOG.Logger)

	a.Assert().NoError(err)
	a.Mocks.API.RDS.AssertExpectations(a.T())
}

func (a *AWSTestSuite) TestTagResourceError() {
	a.Mocks.API.EC2.On("CreateTags", mock.AnythingOfType("*ec2.CreateTagsInput")).Return(nil, errors.New("tag is too long"))
	a.Mocks.LOG.WithFields(logrus.Fields{}).Times(0)

	err := a.Mocks.AWS.TagResource(a.ResourceID, "tag-key", "tag-value", a.Mocks.LOG.Logger)

	a.Assert().Error(err)
	a.Assert().Equal("unable to tag resource id: WSxqXCaZw1dC: tag is too long", err.Error())
	a.Mocks.API.RDS.AssertExpectations(a.T())
}

func (a *AWSTestSuite) TestUntagResource() {
	a.Mocks.API.EC2.On("DeleteTags", mock.AnythingOfType("*ec2.DeleteTagsInput")).Return(&ec2.DeleteTagsOutput{}, nil)
	a.Mocks.LOG.WithFields(logrus.Fields{"tag-key": "tag-key", "tag-value": "tag-value"}).Once()

	err := a.Mocks.AWS.UntagResource(a.ResourceID, "tag-key", "tag-value", a.Mocks.LOG.Logger)

	a.Assert().NoError(err)
	a.Mocks.API.RDS.AssertExpectations(a.T())
}

func (a *AWSTestSuite) TestUntagResourceEmptyResourceID() {
	a.Mocks.API.EC2.On("DeleteTags", mock.AnythingOfType("*ec2.DeleteTagsInput")).Times(0)
	a.Mocks.LOG.WithFields(logrus.Fields{"tag-key": "tag-key", "tag-value": "tag-value"}).Once()

	err := a.Mocks.AWS.UntagResource("", "tag-key", "tag-value", a.Mocks.LOG.Logger)

	a.Assert().Error(err)
	a.Assert().Equal("unable to remove AWS tag from resource: missing resource ID", err.Error())
	a.Mocks.API.RDS.AssertExpectations(a.T())
}

func (a *AWSTestSuite) TestUntagResourceError() {
	a.Mocks.API.EC2.On("DeleteTags", mock.AnythingOfType("*ec2.DeleteTagsInput")).Return(nil, errors.New("tag not found"))
	a.Mocks.LOG.WithFields(logrus.Fields{}).Times(0)

	err := a.Mocks.AWS.UntagResource(a.ResourceID, "tag-key", "tag-value", a.Mocks.LOG.Logger)

	a.Assert().Error(err)
	a.Assert().Equal("unable to remove AWS tag from resource: tag not found", err.Error())
	a.Mocks.API.RDS.AssertExpectations(a.T())
}

func (a *AWSTestSuite) TestIsValidAMIEmptyResourceID() {
	a.Mocks.API.EC2.On("DescribeImages", mock.AnythingOfType("*ec2.DescribeImagesInput")).Times(0)
	a.Mocks.LOG.WithFields(logrus.Fields{}).Times(0)

	ok, err := a.Mocks.AWS.IsValidAMI("")

	a.Assert().NoError(err)
	a.Assert().True(ok)
	a.Mocks.API.RDS.AssertExpectations(a.T())
}

func (a *AWSTestSuite) TestIsValidAMI() {
	a.Mocks.API.EC2.On("DescribeImages", mock.AnythingOfType("*ec2.DescribeImagesInput")).Return(&ec2.DescribeImagesOutput{
		Images: make([]*ec2.Image, 2),
	}, nil).Once()

	ok, err := a.Mocks.AWS.IsValidAMI(a.ResourceID)

	a.Assert().NoError(err)
	a.Assert().True(ok)
	a.Mocks.API.RDS.AssertExpectations(a.T())
}

func (a *AWSTestSuite) TestIsValidAMINoImages() {
	a.Mocks.API.EC2.On("DescribeImages", mock.AnythingOfType("*ec2.DescribeImagesInput")).Return(&ec2.DescribeImagesOutput{
		Images: make([]*ec2.Image, 0),
	}, nil).Once()

	ok, err := a.Mocks.AWS.IsValidAMI(a.ResourceID)

	a.Assert().NoError(err)
	a.Assert().False(ok)
	a.Mocks.API.RDS.AssertExpectations(a.T())
}

func (a *AWSTestSuite) TestIsValidAMIError() {
	a.Mocks.API.EC2.On("DescribeImages", mock.AnythingOfType("*ec2.DescribeImagesInput")).Return(nil, errors.New("resource id not found")).Once()

	ok, err := a.Mocks.AWS.IsValidAMI(a.ResourceID)

	a.Assert().Error(err)
	a.Assert().False(ok)
	a.Assert().Equal("resource id not found", err.Error())
	a.Mocks.API.RDS.AssertExpectations(a.T())
}

func TestVPCReal(t *testing.T) {
	if os.Getenv("SUPER_AWS_VPC_TEST") == "" {
		return
	}

	logger := logrus.New()

	clusterID := "testclusterID1"

	_, err := AWSClient.GetAndClaimVpcResources(clusterID, "testowner", logger)
	require.NoError(t, err)

	err = AWSClient.releaseVpc(clusterID, logger)
	require.NoError(t, err)
}
