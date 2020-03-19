package aws

import (
	"os"
	"sync"
	"testing"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/golang/mock/gomock"
	testlib "github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func (a *AWSTestSuite) TestTagResource() {
	a.Mocks.Log.Logger.EXPECT().
		WithFields(logrus.Fields{"tag-key": "tag-key", "tag-value": "tag-value"}).
		Return(testlib.NewLoggerEntry()).Times(1).
		After(a.Mocks.API.EC2.EXPECT().
			CreateTags(gomock.Any()).
			Return(&ec2.CreateTagsOutput{}, nil))

	err := a.Mocks.AWS.TagResource(a.ResourceID, "tag-key", "tag-value", a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}

func (a *AWSTestSuite) TestTagResourceError() {
	a.Mocks.Log.Logger.EXPECT().
		WithFields(logrus.Fields{"tag-key": "tag-key", "tag-value": "tag-value"}).
		Return(testlib.NewLoggerEntry()).Times(1).
		After(a.Mocks.API.EC2.EXPECT().
			CreateTags(gomock.Any()).
			Return(nil, errors.New("invalid tag")))

	err := a.Mocks.AWS.TagResource(a.ResourceID, "tag-key", "tag-value", a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("unable to tag resource id: WSxqXCaZw1dC: invalid tag", err.Error())
}

func (a *AWSTestSuite) TestUntagResource() {
	a.Mocks.Log.Logger.EXPECT().
		WithFields(logrus.Fields{"tag-key": "tag-key", "tag-value": "tag-value"}).
		Return(testlib.NewLoggerEntry()).Times(1).
		After(a.Mocks.API.EC2.EXPECT().
			DeleteTags(gomock.Any()).
			Return(&ec2.DeleteTagsOutput{}, nil))

	err := a.Mocks.AWS.UntagResource(a.ResourceID, "tag-key", "tag-value", a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}

func (a *AWSTestSuite) TestUntagResourceEmptyResourceID() {
	a.Mocks.Log.Logger.EXPECT().
		WithFields(logrus.Fields{"tag-key": "tag-key", "tag-value": "tag-value"}).
		Return(testlib.NewLoggerEntry()).Times(1).
		After(a.Mocks.API.EC2.EXPECT().
			DeleteTags(gomock.Any()).
			Return(&ec2.DeleteTagsOutput{}, nil))

	err := a.Mocks.AWS.UntagResource("", "tag-key", "tag-value", a.Mocks.Log.Logger)

	a.Assert().Error(err)
	a.Assert().Equal("unable to remove AWS tag from resource: missing resource ID", err.Error())
}

func (a *AWSTestSuite) TestUntagResourceError() {
	a.Mocks.Log.Logger.EXPECT().
		WithFields(logrus.Fields{"tag-key": "tag-key", "tag-value": "tag-value"}).
		Return(testlib.NewLoggerEntry()).Times(1).
		After(a.Mocks.API.EC2.EXPECT().
			DeleteTags(gomock.Any()).
			Return(nil, errors.New("tag not found")))

	err := a.Mocks.AWS.UntagResource(a.ResourceID, "tag-key", "tag-value", a.Mocks.Log.Logger)

	a.Assert().Error(err)
	a.Assert().Equal("unable to remove AWS tag from resource: tag not found", err.Error())
}

func (a *AWSTestSuite) TestIsValidAMIEmptyResourceID() {
	a.Mocks.Log.Logger.EXPECT().
		WithFields(logrus.Fields{}).
		Return(testlib.NewLoggerEntry()).Times(1).
		After(a.Mocks.API.EC2.EXPECT().
			DescribeImages(gomock.Any()).
			Return(nil, errors.New("tag not found")))

	ok, err := a.Mocks.AWS.IsValidAMI("", a.Mocks.Log.Logger)
	a.Assert().NoError(err)
	a.Assert().True(ok)
}

func (a *AWSTestSuite) TestIsValidAMI() {
	a.Mocks.Log.Logger.EXPECT().
		WithFields(logrus.Fields{}).
		Return(testlib.NewLoggerEntry()).Times(1).
		After(a.Mocks.API.EC2.EXPECT().
			DescribeImages(gomock.Any()).
			Return(&ec2.DescribeImagesOutput{
				Images: make([]*ec2.Image, 2),
			}, nil))

	ok, err := a.Mocks.AWS.IsValidAMI(a.ResourceID, a.Mocks.Log.Logger)
	a.Assert().NoError(err)
	a.Assert().True(ok)
}

func (a *AWSTestSuite) TestIsValidAMINoImages() {
	a.Mocks.Log.Logger.EXPECT().
		WithFields(logrus.Fields{}).
		Return(testlib.NewLoggerEntry()).Times(1).
		After(a.Mocks.API.EC2.EXPECT().
			DescribeImages(gomock.Any()).
			Return(&ec2.DescribeImagesOutput{
				Images: make([]*ec2.Image, 0),
			}, nil))

	ok, err := a.Mocks.AWS.IsValidAMI(a.ResourceID, a.Mocks.Log.Logger)

	a.Assert().NoError(err)
	a.Assert().False(ok)
}

func (a *AWSTestSuite) TestIsValidAMIError() {
	a.Mocks.Log.Logger.EXPECT().
		WithFields(logrus.Fields{}).
		Return(testlib.NewLoggerEntry()).Times(1).
		After(a.Mocks.API.EC2.EXPECT().
			DescribeImages(gomock.Any()).
			Return(nil, errors.New("resource id not found")))

	ok, err := a.Mocks.AWS.IsValidAMI(a.ResourceID, log.New())

	a.Assert().Error(err)
	a.Assert().False(ok)
	a.Assert().Equal("resource id not found", err.Error())
}

func TestVPCReal(t *testing.T) {
	if os.Getenv("SUPER_AWS_VPC_TEST") == "" {
		return
	}

	logger := logrus.New()
	client := &Client{
		mux: &sync.Mutex{},
	}

	clusterID := "testclusterID1"

	_, err := client.GetAndClaimVpcResources(clusterID, "testowner", logger)
	require.NoError(t, err)

	err = client.releaseVpc(clusterID, logger)
	require.NoError(t, err)
}
