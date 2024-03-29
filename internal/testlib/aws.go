// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

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
	Route53               *mocks.MockRoute53API
	ResourceGroupsTagging *mocks.MockResourceGroupsTaggingAPIAPI
	SecretsManager        *mocks.MockSecretsManagerAPI
	STS                   *mocks.MockSTSAPI
	DynamoDB              *mocks.MockDynamoDBAPI
}

// NewAWSMockedAPI returns an instance of AWSMockedAPI.
func NewAWSMockedAPI(ctrl *gomock.Controller) *AWSMockedAPI {
	return &AWSMockedAPI{
		ACM:                   mocks.NewMockACMAPI(ctrl),
		RDS:                   mocks.NewMockRDSAPI(ctrl),
		IAM:                   mocks.NewMockIAMAPI(ctrl),
		EC2:                   mocks.NewMockEC2API(ctrl),
		KMS:                   mocks.NewMockKMSAPI(ctrl),
		S3:                    mocks.NewMockS3API(ctrl),
		Route53:               mocks.NewMockRoute53API(ctrl),
		ResourceGroupsTagging: mocks.NewMockResourceGroupsTaggingAPIAPI(ctrl),
		SecretsManager:        mocks.NewMockSecretsManagerAPI(ctrl),
		STS:                   mocks.NewMockSTSAPI(ctrl),
		DynamoDB:              mocks.NewMockDynamoDBAPI(ctrl),
	}
}
