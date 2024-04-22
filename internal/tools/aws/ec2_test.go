// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"github.com/aws/aws-sdk-go-v2/aws"
	"os"
	"sync"
	"testing"

	"github.com/mattermost/mattermost-cloud/model"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/golang/mock/gomock"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
)

func (a *AWSTestSuite) TestTagResource() {
	a.Mocks.Log.Logger.EXPECT().
		WithFields(logrus.Fields{"tag-key": "tag-key", "tag-value": "tag-value"}).
		Return(testlib.NewLoggerEntry()).Times(1).
		After(a.Mocks.API.EC2.EXPECT().
			CreateTags(context.TODO(), gomock.Any()).
			Return(&ec2.CreateTagsOutput{}, nil))

	err := a.Mocks.AWS.TagResource(a.ResourceID, "tag-key", "tag-value", a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}

func (a *AWSTestSuite) TestTagResourceError() {
	a.Mocks.API.EC2.EXPECT().
		CreateTags(context.TODO(), gomock.Any()).
		Return(nil, errors.New("invalid tag"))

	err := a.Mocks.AWS.TagResource(a.ResourceID, "tag-key", "tag-value", a.Mocks.Log.Logger)
	a.Assert().Error(err)
	a.Assert().Equal("unable to tag resource id: WSxqXCaZw1dC: invalid tag", err.Error())
}

func (a *AWSTestSuite) TestUntagResource() {
	a.Mocks.Log.Logger.EXPECT().
		WithFields(logrus.Fields{"tag-key": "tag-key", "tag-value": "tag-value"}).
		Return(testlib.NewLoggerEntry()).Times(1).
		After(a.Mocks.API.EC2.EXPECT().
			DeleteTags(context.TODO(), gomock.Any()).
			Return(&ec2.DeleteTagsOutput{}, nil))

	err := a.Mocks.AWS.UntagResource(a.ResourceID, "tag-key", "tag-value", a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}

func (a *AWSTestSuite) TestUntagResourceEmptyResourceID() {
	err := a.Mocks.AWS.UntagResource("", "tag-key", "tag-value", a.Mocks.Log.Logger)

	a.Assert().Error(err)
	a.Assert().Equal("unable to remove AWS tag from resource: missing resource ID", err.Error())
}

func (a *AWSTestSuite) TestUntagResourceError() {
	a.Mocks.API.EC2.EXPECT().
		DeleteTags(context.TODO(), gomock.Any()).
		Return(nil, errors.New("tag not found"))

	err := a.Mocks.AWS.UntagResource(a.ResourceID, "tag-key", "tag-value", a.Mocks.Log.Logger)

	a.Assert().Error(err)
	a.Assert().Equal("unable to remove AWS tag from resource: tag not found", err.Error())
}

func (a *AWSTestSuite) TestIsValidAMIEmptyResourceID() {
	ok, err := a.Mocks.AWS.IsValidAMI("", a.Mocks.Log.Logger)
	a.Assert().NoError(err)
	a.Assert().True(ok)
}

func (a *AWSTestSuite) TestIsValidAMI() {
	// Test case when AMIImage is a valid AMI ID or AMI name
	testCases := []struct {
		name                  string
		resourceID            string
		expectedDescribeInput *ec2.DescribeImagesInput
	}{
		{
			name:       "Test with AMI ID",
			resourceID: "ami-123456",
			expectedDescribeInput: &ec2.DescribeImagesInput{
				ImageIds: []string{"ami-123456"},
			},
		},
		{
			name:       "Test with AMI Name without architecture suffix",
			resourceID: "example-ami-name",
			expectedDescribeInput: &ec2.DescribeImagesInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("name"),
						Values: []string{"example-ami-name-amd64", "example-ami-name-arm64"},
					},
				},
			},
		},
		{
			name:       "Test with AMI Name with architecture suffix",
			resourceID: "example-ami-name-amd64",
			expectedDescribeInput: &ec2.DescribeImagesInput{
				Filters: []ec2Types.Filter{
					{
						Name:   aws.String("name"),
						Values: []string{"example-ami-name-amd64"},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		a.T().Run(tc.name, func(t *testing.T) {
			a.Mocks.API.EC2.EXPECT().
				DescribeImages(gomock.Any(), gomock.Eq(tc.expectedDescribeInput)).
				Return(&ec2.DescribeImagesOutput{
					Images: []ec2Types.Image{{}, {}}, // Assume 2 images are found
				}, nil)

			ok, err := a.Mocks.AWS.IsValidAMI(tc.resourceID, a.Mocks.Log.Logger)
			a.Assert().NoError(err)
			a.Assert().True(ok)
		})
	}
}

func (a *AWSTestSuite) TestIsValidAMINoImages() {
	// Test case when no images are found
	a.Mocks.API.EC2.EXPECT().
		DescribeImages(gomock.Any(), gomock.Any()).
		Return(&ec2.DescribeImagesOutput{
			Images: []ec2Types.Image{},
		}, nil)

	a.Mocks.Log.Logger.EXPECT().
		Info("No images found matching the criteria", "AMI Names", []string{"example-ami-name-amd64"}).
		Times(1)

	ok, err := a.Mocks.AWS.IsValidAMI("example-ami-name-amd64", a.Mocks.Log.Logger)
	a.Assert().NoError(err)
	a.Assert().False(ok)
}

func (a *AWSTestSuite) TestIsValidAMIError() {
	errorMsg := errors.New("resource id not found")

	// Assuming testlib.NewLoggerEntry() returns a mock or a stub that satisfies the logger's interface
	mockEntry := testlib.NewLoggerEntry()

	// Setup the expectation for WithError, expecting any error and returning the mock logger entry
	a.Mocks.Log.Logger.EXPECT().WithError(gomock.Any()).Return(mockEntry).AnyTimes()

	// Setup the EC2 mock to return an error for DescribeImages
	a.Mocks.API.EC2.EXPECT().
		DescribeImages(gomock.Any(), gomock.Any()).
		Return(nil, errorMsg).Times(1)

	// Call the function under test
	ok, err := a.Mocks.AWS.IsValidAMI("ami-failingcase", a.Mocks.Log.Logger)

	// Assert that an error was returned as expected
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

	cluster := &model.Cluster{ID: "testclusterID1"}

	_, err := client.GetAndClaimVpcResources(cluster, "testowner", logger)
	require.NoError(t, err)

	err = client.releaseVpc(cluster, logger)
	require.NoError(t, err)
}
