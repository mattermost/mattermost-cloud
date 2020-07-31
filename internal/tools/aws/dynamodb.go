// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/dynamodb"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// DynamoDBEnsureTableDeleted is used to check if DynamoDB table exists and delete it.
func (a *Client) DynamoDBEnsureTableDeleted(tableName string, logger log.FieldLogger) error {
	// First check if table still exists.
	_, err := a.Service().dynamodb.DescribeTable(&dynamodb.DescribeTableInput{
		TableName: aws.String(tableName),
	})
	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() == dynamodb.ErrCodeResourceNotFoundException {
			logger.WithField("dynamodb-table", tableName).Warn("DynamoDB table could not be found; assuming already deleted")
			return nil
		}
	}

	_, err = a.Service().dynamodb.DeleteTable(&dynamodb.DeleteTableInput{
		TableName: aws.String(tableName),
	})
	if err != nil {
		return errors.Wrap(err, "unable to delete bucket")
	}

	return nil
}
