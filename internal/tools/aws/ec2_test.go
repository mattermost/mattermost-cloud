package aws

import (
	"errors"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func (api *mockAPI) getEC2Client() (*ec2.EC2, error) {
	return nil, api.returnedError
}

func (api *mockAPI) getACMClient() (*acm.ACM, error) {
	return nil, api.returnedError
}

func (api *mockAPI) tagResource(svc *ec2.EC2, input *ec2.CreateTagsInput) (*ec2.CreateTagsOutput, error) {
	return nil, api.returnedError
}

func (api *mockAPI) untagResource(svc *ec2.EC2, input *ec2.DeleteTagsInput) (*ec2.DeleteTagsOutput, error) {
	return nil, api.returnedError
}

func (api *mockAPI) describeImages(svc *ec2.EC2, input *ec2.DescribeImagesInput) (*ec2.DescribeImagesOutput, error) {
	return api.returnedDescribeImagesOutput, api.returnedError
}

func TestTagResource(t *testing.T) {
	tests := []struct {
		name        string
		resourceID  string
		key         string
		value       string
		mockError   error
		expectError bool
	}{
		{
			"set tag",
			"resource1",
			"key1",
			"value1",
			nil,
			false,
		},
		{
			"missing resource ID",
			"",
			"key1",
			"value1",
			nil,
			true,
		},
		{
			"bad resource ID",
			"badid",
			"key1",
			"value1",
			errors.New("mock bad resource id"),
			true,
		},
	}

	logger := logrus.New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Client{
				api: &mockAPI{returnedError: tt.mockError},
			}

			err := a.TagResource(tt.resourceID, tt.key, tt.value, logger)
			switch tt.expectError {
			case true:
				assert.Error(t, err)
			case false:
				assert.NoError(t, err)
			}
		})
	}
}

func TestUntagResource(t *testing.T) {
	tests := []struct {
		name        string
		resourceID  string
		key         string
		value       string
		mockError   error
		expectError bool
	}{
		{
			"unset tag",
			"resource1",
			"key1",
			"value1",
			nil,
			false,
		},
		{
			"missing resource ID",
			"",
			"key1",
			"value1",
			nil,
			true,
		},
	}

	logger := logrus.New()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Client{
				api: &mockAPI{returnedError: tt.mockError},
			}

			err := a.UntagResource(tt.resourceID, tt.key, tt.value, logger)
			switch tt.expectError {
			case true:
				assert.Error(t, err)
			case false:
				assert.NoError(t, err)
			}
		})
	}
}

func TestVPCReal(t *testing.T) {
	if os.Getenv("SUPER_AWS_VPC_TEST") == "" {
		return
	}

	logger := logrus.New()
	awsClient := New()

	clusterID := "testclusterID1"

	_, err := awsClient.GetAndClaimVpcResources(clusterID, logger)
	require.NoError(t, err)

	err = awsClient.releaseVpc(clusterID, logger)
	require.NoError(t, err)
}

func TestAMIs(t *testing.T) {
	tests := []struct {
		name           string
		AMIID          string
		mockError      error
		mockResponse   *ec2.DescribeImagesOutput
		expectResponse bool
	}{
		{
			name:           "invalid AMI",
			AMIID:          "invalid-AMI-ID",
			mockError:      errors.New("invalid AMI ID"),
			mockResponse:   &ec2.DescribeImagesOutput{},
			expectResponse: false,
		},
		{
			name:      "valid AMI",
			AMIID:     "valid-AMI-ID",
			mockError: nil,
			mockResponse: &ec2.DescribeImagesOutput{
				Images: []*ec2.Image{
					{
						ImageId:      aws.String("valid-AMI-ID"),
						CreationDate: aws.String("2020-01-13T00:00:00.001Z"),
					},
				},
			},
			expectResponse: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := Client{
				api: &mockAPI{returnedDescribeImagesOutput: tt.mockResponse, returnedError: tt.mockError},
			}

			isValid, err := a.IsValidAMI(tt.AMIID)
			switch tt.expectResponse {
			case true:
				assert.True(t, isValid)
				assert.Nil(t, err)
			case false:
				assert.False(t, isValid)
				assert.NotNil(t, err)
			}
		})
	}
}
