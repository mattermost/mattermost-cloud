// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	gt "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	gtTypes "github.com/aws/aws-sdk-go-v2/service/resourcegroupstaggingapi/types"
	"github.com/aws/aws-sdk-go/aws/awserr"

	"github.com/golang/mock/gomock"
)

func (a *AWSTestSuite) TestResourceTaggingGetAllResources() {
	gomock.InOrder(
		a.Mocks.API.ResourceGroupsTagging.EXPECT().
			GetResources(gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, input *gt.GetResourcesInput, optFns ...func(*gt.Options)) {
				a.Assert().Equal(DefaultRDSEncryptionTagKey, *input.TagFilters[0].Key)
				a.Assert().Equal(CloudID(a.InstallationA.ID), input.TagFilters[0].Values[0])
				a.Assert().Nil(input.PaginationToken)
			}).
			Return(&gt.GetResourcesOutput{
				PaginationToken: aws.String("next_token"),
				ResourceTagMappingList: []gtTypes.ResourceTagMapping{
					{
						ResourceARN: aws.String(a.ResourceARN),
					},
				},
			}, nil).
			Times(1),

		a.Mocks.API.ResourceGroupsTagging.EXPECT().
			GetResources(gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, input *gt.GetResourcesInput, optFns ...func(*gt.Options)) {
				a.Assert().Equal(DefaultRDSEncryptionTagKey, *input.TagFilters[0].Key)
				a.Assert().Equal(CloudID(a.InstallationA.ID), input.TagFilters[0].Values[0])
				a.Assert().Equal("next_token", *input.PaginationToken)
			}).
			Return(&gt.GetResourcesOutput{
				PaginationToken: nil,
				ResourceTagMappingList: []gtTypes.ResourceTagMapping{
					{
						ResourceARN: &a.ResourceARN,
					},
				},
			}, nil).
			Times(1),
	)

	result, err := a.Mocks.AWS.resourceTaggingGetAllResources(gt.GetResourcesInput{
		TagFilters: []gtTypes.TagFilter{
			{
				Key:    aws.String(DefaultRDSEncryptionTagKey),
				Values: []string{CloudID(a.InstallationA.ID)},
			},
		},
	})

	a.Assert().NoError(err)
	a.Assert().Equal(2, len(result))
}

func (a *AWSTestSuite) TestResourceTaggingGetAllResourcesEmpty() {
	gomock.InOrder(
		a.Mocks.API.ResourceGroupsTagging.EXPECT().
			GetResources(gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, input *gt.GetResourcesInput, optFns ...func(*gt.Options)) {
				a.Assert().Equal(DefaultRDSEncryptionTagKey, *input.TagFilters[0].Key)
				a.Assert().Equal(CloudID(a.InstallationA.ID), input.TagFilters[0].Values[0])
				a.Assert().Nil(input.PaginationToken)
			}).
			Return(&gt.GetResourcesOutput{}, nil).
			Times(1),
	)

	result, err := a.Mocks.AWS.resourceTaggingGetAllResources(gt.GetResourcesInput{
		TagFilters: []gtTypes.TagFilter{
			{
				Key:    aws.String(DefaultRDSEncryptionTagKey),
				Values: []string{CloudID(a.InstallationA.ID)},
			},
		},
	})

	a.Assert().NoError(err)
	a.Assert().Equal(0, len(result))
}

func (a *AWSTestSuite) TestResourceTaggingGetAllResourcesError() {
	gomock.InOrder(
		a.Mocks.API.ResourceGroupsTagging.EXPECT().
			GetResources(gomock.Any(), gomock.Any()).
			Do(func(ctx context.Context, input *gt.GetResourcesInput, optFns ...func(*gt.Options)) {
				a.Assert().Equal(DefaultRDSEncryptionTagKey, *input.TagFilters[0].Key)
				a.Assert().Equal(CloudID(a.InstallationA.ID), input.TagFilters[0].Values[0])
				a.Assert().Nil(input.PaginationToken)
			}).
			Return(nil, awserr.New("InternalServerError", "something went wrong", nil)).
			Times(1),
	)

	result, err := a.Mocks.AWS.resourceTaggingGetAllResources(gt.GetResourcesInput{
		TagFilters: []types.TagFilter{
			{
				Key:    aws.String(DefaultRDSEncryptionTagKey),
				Values: []string{CloudID(a.InstallationA.ID)},
			},
		},
	})

	a.Assert().Nil(result)
	a.Assert().Error(err)
	a.Assert().Equal("InternalServerError: something went wrong", err.Error())
}
