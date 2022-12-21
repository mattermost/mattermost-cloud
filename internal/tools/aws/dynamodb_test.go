// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/golang/mock/gomock"
)

func (a *AWSTestSuite) TestDynamoDBEnsureTableDeleted() {
	gomock.InOrder(
		a.Mocks.API.DynamoDB.EXPECT().
			DescribeTable(gomock.Any(), gomock.Any()).
			Return(&dynamodb.DescribeTableOutput{}, nil),

		a.Mocks.API.DynamoDB.EXPECT().
			DeleteTable(gomock.Any(), gomock.Any()).
			Return(&dynamodb.DeleteTableOutput{}, nil),
	)

	err := a.Mocks.AWS.DynamoDBEnsureTableDeleted("table-name", a.Mocks.Log.Logger)
	a.Assert().NoError(err)
}
