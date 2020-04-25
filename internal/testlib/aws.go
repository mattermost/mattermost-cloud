package testlib

import (
	"github.com/golang/mock/gomock"
	mocks "github.com/mattermost/mattermost-cloud/internal/mocks/aws-sdk"
)

// AWSMockedAPI has all AWS mocked services. New services should be added here.
type AWSMockedAPI struct {
	ACM                   *mocks.MockACMAPI
	RDS                   *mocks.MockRDSAPI
	IAM                   *mocks.MockIAMAPI
	EC2                   *mocks.MockEC2API
	KMS                   *mocks.MockKMSAPI
	S3                    *mocks.MockS3API
	STS                   *mocks.MockSTSAPI
	Route53               *mocks.MockRoute53API
	ResourceGroupsTagging *mocks.MockResourceGroupsTaggingAPIAPI
	SecretsManager        *mocks.MockSecretsManagerAPI
}

// NewAWSMockedAPI returns an instance of AWSMockedAPI.
func NewAWSMockedAPI(ctrl *gomock.Controller) *AWSMockedAPI {
	return &AWSMockedAPI{
		ACM:                   mocks.NewMockACMAPI(ctrl),
		RDS:                   mocks.NewMockRDSAPI(ctrl),
		IAM:                   mocks.NewMockIAMAPI(ctrl),
		EC2:                   mocks.NewMockEC2API(ctrl),
		KMS:                   mocks.NewMockKMSAPI(ctrl),
		STS:                   mocks.NewMockSTSAPI(ctrl),
		S3:                    mocks.NewMockS3API(ctrl),
		Route53:               mocks.NewMockRoute53API(ctrl),
		ResourceGroupsTagging: mocks.NewMockResourceGroupsTaggingAPIAPI(ctrl),
		SecretsManager:        mocks.NewMockSecretsManagerAPI(ctrl),
	}
}
