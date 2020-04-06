package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/pkg/errors"
)

// kmsCreateSymmetricKey creates a symmetric encryption key with alias.
func (a *Client) kmsCreateSymmetricKeyWithAlias(awsID, aliasName, keyDescription string) (*kms.KeyMetadata, error) {
	createKeyOut, err := a.Service().kms.CreateKey(&kms.CreateKeyInput{
		Description: aws.String(keyDescription),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unabled to create encryption key for %s", awsID)
	}

	// Allows retrieving the key ID in one call and without special permissions that would be necessary if
	// we were using tags.
	_, err = a.Service().kms.CreateAlias(&kms.CreateAliasInput{
		AliasName:   aws.String(aliasName),
		TargetKeyId: createKeyOut.KeyMetadata.KeyId,
	})
	if err != nil {
		return createKeyOut.KeyMetadata, errors.Wrapf(err, "unabled to create an alias name for encryption key %s", *createKeyOut.KeyMetadata.Arn)
	}

	return createKeyOut.KeyMetadata, nil
}

// kmsDisableSymmetricKey disable a symmetric encryption key with alias.
func (a *Client) kmsDisableSymmetricKey(keyID string) error {
	_, err := a.Service().kms.DisableKey(&kms.DisableKeyInput{
		KeyId: aws.String(keyID),
	})
	if err != nil {
		return errors.Wrapf(err, "unabled to disable encryption key %s", keyID)
	}

	return nil
}

// kmsGetSymmetricKey get a symmetric encryption key with alias.
func (a *Client) kmsGetSymmetricKey(aliasName string) (*kms.KeyMetadata, error) {
	describeKeyOut, err := a.Service().kms.DescribeKey(&kms.DescribeKeyInput{
		KeyId: aws.String(aliasName),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unabled to describe encryption key for alias name %s", aliasName)
	}

	return describeKeyOut.KeyMetadata, nil
}

// kmsScheduleKeyDeletion sets a supplied key for deletion in n days. If days is -1, this
// method will set deletion to a contant default value. See tools/aws/contants.go
func (a *Client) kmsScheduleKeyDeletion(keyID string, days int64) error {
	input := kms.ScheduleKeyDeletionInput{
		KeyId: aws.String(keyID),
	}

	if days == -1 {
		input.PendingWindowInDays = aws.Int64(DefaultScheduledEncryptionKeyDeletion)
	} else {
		input.PendingWindowInDays = aws.Int64(days)
	}

	_, err := a.Service().kms.ScheduleKeyDeletion(&input)
	if err != nil {
		return errors.Wrapf(err, "unabled to describe encryption key ID %s", keyID)
	}

	return nil
}
