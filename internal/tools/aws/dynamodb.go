// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// DynamoDBEnsureTableDeleted is used to check if DynamoDB table exists and delete it.
func (a *Client) DynamoDBEnsureTableDeleted(tableName string, logger log.FieldLogger) error {
	// First check if table still exists.
	_, err := a.Service().dynamodb.DescribeTable(
		context.TODO(),
		&dynamodb.DescribeTableInput{
			TableName: aws.String(tableName),
		})

	if err != nil {
		var awsErr *types.ResourceNotFoundException
		if errors.As(err, &awsErr) {
			logger.WithField("dynamodb-table", tableName).Warn("DynamoDB table could not be found; assuming already deleted")
			return nil
		}
		logger.WithField("dynamodb-table", tableName).WithError(err).Warn("Error checking for dynamodb table")
	}

	_, err = a.Service().dynamodb.DeleteTable(
		context.TODO(),
		&dynamodb.DeleteTableInput{
			TableName: aws.String(tableName),
		})
	if err != nil {
		return errors.Wrap(err, "unable to delete DynamoDB table")
	}

	return nil
}
