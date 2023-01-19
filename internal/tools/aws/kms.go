// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmsTypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
)

// kmsCreateSymmetricKey creates a symmetric encryption key with alias.
func (a *Client) kmsCreateSymmetricKey(keyDescription string, tags []kmsTypes.Tag) (*kmsTypes.KeyMetadata, error) {
	createKeyOut, err := a.Service().kms.CreateKey(
		context.TODO(),
		&kms.CreateKeyInput{
			Description: aws.String(keyDescription),
			Tags:        tags,
		})
	if err != nil {
		return nil, err
	}

	return createKeyOut.KeyMetadata, nil
}

// kmsGetSymmetricKey get a symmetric encryption key with alias.
func (a *Client) kmsGetSymmetricKey(aliasName string) (*kmsTypes.KeyMetadata, error) {
	describeKeyOut, err := a.Service().kms.DescribeKey(
		context.TODO(),
		&kms.DescribeKeyInput{
			KeyId: aws.String(aliasName),
		})
	if err != nil {
		return nil, err
	}

	return describeKeyOut.KeyMetadata, nil
}

// kmsScheduleKeyDeletion sets a supplied key for deletion in n days. The service will return an error
// if scheduled time is are less than 7 or more than 30 days.
// https://docs.aws.amazon.com/kms/latest/APIReference/API_ScheduleKeyDeletion.html#API_ScheduleKeyDeletion_RequestSyntax
func (a *Client) kmsScheduleKeyDeletion(keyID string, days int32) error {
	_, err := a.Service().kms.ScheduleKeyDeletion(
		context.TODO(),
		&kms.ScheduleKeyDeletionInput{
			KeyId:               aws.String(keyID),
			PendingWindowInDays: aws.Int32(days),
		})
	if err != nil {
		return err
	}

	return nil
}

// kmsEncrypt encrypts a given value with the provided KMS key.
func (a *Client) kmsEncrypt(keyID, value string) ([]byte, error) {
	enc, err := a.Service().kms.Encrypt(
		context.TODO(),
		&kms.EncryptInput{
			KeyId:     aws.String(keyID),
			Plaintext: []byte(value),
		})
	if err != nil {
		return nil, err
	}

	return enc.CiphertextBlob, nil
}
