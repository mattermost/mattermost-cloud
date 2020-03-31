package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/pkg/errors"
)

// kmsCreateSymmetricKey creates a symmetric encryption key with alias.
func (a *Client) kmsCreateSymmetricKey(awsID, aliasName, keyDescription string) (*kms.KeyMetadata, error) {
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
		return nil, errors.Wrapf(err, "unabled to create an alias name for encryption key %s", *createKeyOut.KeyMetadata.Arn)
	}

	return createKeyOut.KeyMetadata, nil
}
