package aws

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// IAMAccessKey is the ID and Secret of an AWS IAM user's access key.
type IAMAccessKey struct {
	ID     string
	Secret string
}

func (a *Client) secretsManagerEnsureIAMAccessKeySecretCreated(awsID string, ak *iam.AccessKey) error {
	svc := secretsmanager.New(session.New())

	accessKeyPayload := &IAMAccessKey{
		ID:     *ak.AccessKeyId,
		Secret: *ak.SecretAccessKey,
	}
	b, err := json.Marshal(&accessKeyPayload)
	if err != nil {
		return errors.Wrap(err, "unable to marshal secrets manager payload")
	}

	_, err = svc.CreateSecret(&secretsmanager.CreateSecretInput{
		Name:         aws.String(awsID),
		Description:  aws.String(fmt.Sprintf("IAM access key for user %s", awsID)),
		SecretString: aws.String(string(b)),
	})
	if err != nil {
		return errors.Wrap(err, "unable to create secrets manager secret")
	}

	return nil
}

func secretsManagerGetIAMAccessKey(awsID string) (*IAMAccessKey, error) {
	svc := secretsmanager.New(session.New())

	result, err := svc.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(awsID),
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to get secrets manager secret")
	}

	var aimAccessKey *IAMAccessKey
	err = json.Unmarshal([]byte(*result.SecretString), &aimAccessKey)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal secrets manager payload")
	}

	if aimAccessKey.ID == "" {
		return nil, errors.New("access key ID value was empty")
	}
	if aimAccessKey.Secret == "" {
		return nil, errors.New("access key secret value was empty")
	}

	return aimAccessKey, nil
}

func (a *Client) secretsManagerEnsureIAMAccessKeySecretDeleted(awsID string, logger log.FieldLogger) error {
	svc := secretsmanager.New(session.New())

	_, err := svc.DeleteSecret(&secretsmanager.DeleteSecretInput{
		SecretId: aws.String(awsID),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == secretsmanager.ErrCodeResourceNotFoundException {
				return nil
			}
			return err
		}
		return err
	}

	return nil
}
