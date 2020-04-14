package testlib

import (
	"github.com/golang/mock/gomock"
	mocks "github.com/mattermost/mattermost-cloud/internal/mocks/aws-sdk"
)

// AWSMockedAPI has all AWS mocked services. New services should be added here.
type AWSMockedAPI struct {
	ACM            *mocks.MockACMAPI
	RDS            *mocks.MockRDSAPI
	IAM            *mocks.MockIAMAPI
	EC2            *mocks.MockEC2API
	S3             *mocks.MockS3API
	Route53        *mocks.MockRoute53API
	SecretsManager *mocks.MockSecretsManagerAPI
	KMS            *mocks.MockKMSAPI
}

// NewAWSMockedAPI returns an instance of AWSMockedAPI.
func NewAWSMockedAPI(ctrl *gomock.Controller) *AWSMockedAPI {
	return &AWSMockedAPI{
		ACM:            mocks.NewMockACMAPI(ctrl),
		RDS:            mocks.NewMockRDSAPI(ctrl),
		IAM:            mocks.NewMockIAMAPI(ctrl),
		EC2:            mocks.NewMockEC2API(ctrl),
		S3:             mocks.NewMockS3API(ctrl),
		Route53:        mocks.NewMockRoute53API(ctrl),
		SecretsManager: mocks.NewMockSecretsManagerAPI(ctrl),
		KMS:            mocks.NewMockKMSAPI(ctrl),
	}
}
