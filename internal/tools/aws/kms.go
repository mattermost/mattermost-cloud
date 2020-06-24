// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
)

// kmsCreateSymmetricKey creates a symmetric encryption key with alias.
func (a *Client) kmsCreateSymmetricKey(keyDescription string, tags []*kms.Tag) (*kms.KeyMetadata, error) {
	createKeyOut, err := a.Service().kms.CreateKey(&kms.CreateKeyInput{
		Description: aws.String(keyDescription),
		Tags:        tags,
	})
	if err != nil {
		return nil, err
	}

	return createKeyOut.KeyMetadata, nil
}

// kmsCreateAlias creates an alias for a symmetric encryption key. Alias allows retrieving the key ID in one call and
// without special permissions that would be necessary if looking up it by tags for example.
func (a *Client) kmsCreateAlias(keyID, aliasName string) error {
	_, err := a.Service().kms.CreateAlias(&kms.CreateAliasInput{
		AliasName:   aws.String(aliasName),
		TargetKeyId: aws.String(keyID),
	})
	if err != nil {
		return err
	}

	return nil
}

// kmsDisableSymmetricKey disable a symmetric encryption key with alias.
func (a *Client) kmsDisableSymmetricKey(keyID string) error {
	_, err := a.Service().kms.DisableKey(&kms.DisableKeyInput{
		KeyId: aws.String(keyID),
	})
	if err != nil {
		return err
	}

	return nil
}

// kmsGetSymmetricKey get a symmetric encryption key with alias.
func (a *Client) kmsGetSymmetricKey(aliasName string) (*kms.KeyMetadata, error) {
	describeKeyOut, err := a.Service().kms.DescribeKey(&kms.DescribeKeyInput{
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
func (a *Client) kmsScheduleKeyDeletion(keyID string, days int64) error {
	_, err := a.Service().kms.ScheduleKeyDeletion(&kms.ScheduleKeyDeletionInput{
		KeyId:               aws.String(keyID),
		PendingWindowInDays: aws.Int64(days),
	})
	if err != nil {
		return err
	}

	return nil
}
