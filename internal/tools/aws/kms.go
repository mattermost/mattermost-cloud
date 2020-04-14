package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/pkg/errors"
)

func (a *Client) KmsCreateSymmetricKey(awsID, keyDescription string) (*kms.KeyMetadata, error) {
	return a.kmsCreateSymmetricKey(awsID, keyDescription)
}

// kmsCreateSymmetricKey creates a symmetric encryption key with alias.
func (a *Client) kmsCreateSymmetricKey(awsID, keyDescription string) (*kms.KeyMetadata, error) {
	createKeyOut, err := a.Service().kms.CreateKey(&kms.CreateKeyInput{
		Description: aws.String(keyDescription),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unabled to create encryption key for %s", awsID)
	}

	return createKeyOut.KeyMetadata, nil
}

func (a *Client) KmsCreateAlias(keyID, aliasName string) error {
	return a.kmsCreateAlias(keyID, aliasName)
}

// kmsCreateAlias creates an alias for a symmetric encryption key. Alias allows retrieving the key ID in one call and
// without special permissions that would be necessary if looking up it by tags for example.
func (a *Client) kmsCreateAlias(keyID, aliasName string) error {
	_, err := a.Service().kms.CreateAlias(&kms.CreateAliasInput{
		AliasName:   aws.String(aliasName),
		TargetKeyId: aws.String(keyID),
	})
	if err != nil {
		return errors.Wrapf(err, "unabled to create an alias name for encryption key %s", keyID)
	}

	return nil
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
func (a *Client) KmsGetSymmetricKey(aliasName string) (*kms.KeyMetadata, error) {
	return a.kmsGetSymmetricKey(aliasName)
}

func (a *Client) kmsGetSymmetricKey(aliasName string) (*kms.KeyMetadata, error) {
	describeKeyOut, err := a.Service().kms.DescribeKey(&kms.DescribeKeyInput{
		KeyId: aws.String(aliasName),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unabled to describe encryption key for alias name %s", aliasName)
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
		return errors.Wrapf(err, "unable to schedule deletion of encryption key ID %s", keyID)
	}

	return nil
}

func (a *Client) KmsScheduleKeyDeletion(keyID string, days int64) error {
	return a.kmsScheduleKeyDeletion(keyID, days)
}
