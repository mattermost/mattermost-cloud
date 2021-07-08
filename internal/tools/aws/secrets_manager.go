// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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

// Validate performs a basic sanity check on the IAM Access Key secret.
func (s *IAMAccessKey) Validate() error {
	if s.ID == "" {
		return errors.New("Access key ID value is empty")
	}
	if s.Secret == "" {
		return errors.New("Access key secret value is empty")
	}

	return nil
}

// RDSSecret is the Secret payload for RDS configuration.
type RDSSecret struct {
	MasterUsername string
	MasterPassword string
}

// Validate performs a basic sanity check on the RDS secret.
func (s *RDSSecret) Validate() error {
	if s.MasterUsername == "" {
		return errors.New("RDS master username value is empty")
	}
	if s.MasterPassword == "" {
		return errors.New("RDS master password value is empty")
	}
	if len(s.MasterPassword) != 40 {
		return errors.New("RDS master password length should be equal to 40")
	}

	return nil
}

// SecretsManagerRestoreSecret restores a deleted secret.
func (a *Client) SecretsManagerRestoreSecret(secretName string, logger log.FieldLogger) error {
	_, err := a.Service().secretsManager.RestoreSecret(&secretsmanager.RestoreSecretInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == secretsmanager.ErrCodeResourceNotFoundException {
				logger.WithField("secret-name", secretName).Warn("Secret Manager secret could not be found; assuming fully deleted")

				return nil
			}
		}
		return err
	}

	logger.WithField("secret-name", secretName).Debug("Secret Manager secret recovered")

	return nil
}

// SecretsManagerGetPGBouncerAuthUserPassword returns the pgbouncer auth user password.
func (a *Client) SecretsManagerGetPGBouncerAuthUserPassword(vpcID string) (string, error) {
	authUserSecretName := PGBouncerAuthUserSecretName(vpcID)

	result, err := a.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(PGBouncerAuthUserSecretName(vpcID)),
	})
	if err != nil {
		return "", errors.Wrapf(err, "failed to get pgbouncer auth user secret %s", authUserSecretName)
	}
	secret, err := unmarshalSecretPayload(*result.SecretString)
	if err != nil {
		return "", errors.Wrap(err, "failed to unmarshal secret payload")
	}

	return secret.MasterPassword, nil
}

func (a *Client) secretsManagerEnsureIAMAccessKeySecretCreated(awsID string, ak *iam.AccessKey, logger log.FieldLogger) error {
	accessKeyPayload := &IAMAccessKey{
		ID:     *ak.AccessKeyId,
		Secret: *ak.SecretAccessKey,
	}
	err := accessKeyPayload.Validate()
	if err != nil {
		return err
	}

	b, err := json.Marshal(&accessKeyPayload)
	if err != nil {
		return errors.Wrap(err, "unable to marshal secrets manager payload")
	}

	secretName := IAMSecretName(awsID)
	_, err = a.Service().secretsManager.CreateSecret(&secretsmanager.CreateSecretInput{
		Name:         aws.String(secretName),
		Description:  aws.String(fmt.Sprintf("IAM access key for user %s", awsID)),
		SecretString: aws.String(string(b)),
	})
	if err != nil {
		return errors.Wrap(err, "unable to create secrets manager secret")
	}

	return nil
}

func (a *Client) secretsManagerEnsureRDSSecretCreated(awsID string, logger log.FieldLogger) (*RDSSecret, error) {
	secretName := RDSSecretName(awsID)
	rdsSecretPayload := &RDSSecret{}

	// Check if we already have an RDS secret for this installation.
	result, err := a.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err == nil {
		logger.WithField("secret-name", secretName).Debug("AWS RDS secret already created")

		err = json.Unmarshal([]byte(*result.SecretString), &rdsSecretPayload)
		if err != nil {
			return nil, errors.Wrap(err, "unable to marshal secrets manager payload")
		}

		err := rdsSecretPayload.Validate()
		if err != nil {
			return nil, err
		}

		return rdsSecretPayload, nil
	}

	// There is no existing secret, so we will create a new one with a strong
	// random username and password.
	rdsSecretPayload.MasterUsername = DefaultMattermostDatabaseUsername
	rdsSecretPayload.MasterPassword = newRandomPassword(40)
	err = rdsSecretPayload.Validate()
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(&rdsSecretPayload)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal secrets manager payload")
	}

	_, err = a.Service().secretsManager.CreateSecret(&secretsmanager.CreateSecretInput{
		Name:         aws.String(secretName),
		Description:  aws.String(fmt.Sprintf("RDS configuration for %s", awsID)),
		SecretString: aws.String(string(b)),
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to create secrets manager secret")
	}

	logger.WithField("secret-name", secretName).Debug("AWS RDS secret created")

	return rdsSecretPayload, nil
}

// secretsManagerGetIAMAccessKey returns the AccessKey for an IAM account.
func (a *Client) secretsManagerGetIAMAccessKey(awsID string) (*IAMAccessKey, error) {
	return a.secretsManagerGetIAMAccessKeyFromSecretName(IAMSecretName(awsID))
}

// secretsManagerGetIAMAccessKeyFromSecretName attempts to parse a secret into
// and IAMAccessKey.
func (a *Client) secretsManagerGetIAMAccessKeyFromSecretName(secretName string) (*IAMAccessKey, error) {
	result, err := a.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to get secrets manager secret")
	}

	var iamAccessKey *IAMAccessKey
	err = json.Unmarshal([]byte(*result.SecretString), &iamAccessKey)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal secrets manager payload")
	}

	err = iamAccessKey.Validate()
	if err != nil {
		return nil, err
	}

	return iamAccessKey, nil
}

func (a *Client) secretsManagerGetRDSSecret(awsID string, logger log.FieldLogger) (*RDSSecret, error) {
	secretName := RDSSecretName(awsID)
	result, err := a.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to get secrets manager secret")
	}

	var rdsSecret *RDSSecret
	err = json.Unmarshal([]byte(*result.SecretString), &rdsSecret)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal secrets manager payload")
	}

	err = rdsSecret.Validate()
	if err != nil {
		return nil, err
	}

	return rdsSecret, nil
}

func (a *Client) secretsManagerEnsureIAMAccessKeySecretDeleted(awsID string, logger log.FieldLogger) error {
	return a.secretsManagerEnsureSecretDeleted(IAMSecretName(awsID), logger)
}

func (a *Client) secretsManagerEnsureRDSSecretDeleted(awsID string, logger log.FieldLogger) error {
	return a.secretsManagerEnsureSecretDeleted(RDSSecretName(awsID), logger)
}

func (a *Client) secretsManagerEnsureSecretDeleted(secretName string, logger log.FieldLogger) error {
	_, err := a.Service().secretsManager.DeleteSecret(&secretsmanager.DeleteSecretInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == secretsmanager.ErrCodeResourceNotFoundException {
				logger.WithField("secret-name", secretName).Warn("Secret Manager secret could not be found; assuming already deleted")

				return nil
			}
		}
		return err
	}

	logger.WithField("secret-name", secretName).Debug("Secret Manager secret deleted")

	return nil
}

func unmarshalSecretPayload(payload string) (*RDSSecret, error) {
	var secret RDSSecret
	err := json.Unmarshal([]byte(payload), &secret)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal secrets manager payload")
	}

	err = secret.Validate()
	if err != nil {
		return nil, err
	}

	return &secret, nil
}
