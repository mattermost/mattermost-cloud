// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	iamTypes "github.com/aws/aws-sdk-go-v2/service/iam/types"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager/types"
	"github.com/mattermost/mattermost-cloud/model"
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
	if len(s.MasterPassword) != model.DefaultPasswordLength {
		return errors.New(fmt.Sprintf("RDS master password length should be equal to %d", model.DefaultPasswordLength))
	}

	return nil
}

// SecretsManagerRestoreSecret restores a deleted secret.
func (a *Client) SecretsManagerRestoreSecret(secretName string, logger log.FieldLogger) error {
	_, err := a.Service().secretsManager.RestoreSecret(
		context.TODO(),
		&secretsmanager.RestoreSecretInput{
			SecretId: aws.String(secretName),
		})
	if err != nil {
		var awsErr *types.ResourceNotFoundException
		if errors.As(err, &awsErr) {
			logger.WithField("secret-name", secretName).Warn("Secret Manager secret could not be found; assuming fully deleted")
			return nil
		}
		return err
	}

	logger.WithField("secret-name", secretName).Debug("Secret Manager secret recovered")

	return nil
}

// SecretsManagerGetPGBouncerAuthUserPassword returns the pgbouncer auth user password.
func (a *Client) SecretsManagerGetPGBouncerAuthUserPassword(vpcID string) (string, error) {
	authUserSecretName := PGBouncerAuthUserSecretName(vpcID)

	secret, err := a.secretsManagerGetRDSSecret(authUserSecretName)
	if err != nil {
		return "", errors.Wrap(err, "failed to get secret")
	}

	return secret.MasterPassword, nil
}

func (a *Client) secretsManagerEnsureIAMAccessKeySecretCreated(awsID string, ak *iamTypes.AccessKey) error {
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
	_, err = a.Service().secretsManager.CreateSecret(
		context.TODO(),
		&secretsmanager.CreateSecretInput{
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
	result, err := a.Service().secretsManager.GetSecretValue(
		context.TODO(),
		&secretsmanager.GetSecretValueInput{
			SecretId: aws.String(secretName),
		})
	if err == nil {
		logger.WithField("secret-name", secretName).Debug("AWS RDS secret already created")

		err = json.Unmarshal([]byte(*result.SecretString), &rdsSecretPayload)
		if err != nil {
			return nil, errors.Wrap(err, "unable to marshal secrets manager payload")
		}

		err = rdsSecretPayload.Validate()
		if err != nil {
			return nil, err
		}

		return rdsSecretPayload, nil
	}

	// There is no existing secret, so we will create a new one with a strong
	// random username and password.
	rdsSecretPayload.MasterUsername = model.DefaultMattermostDatabaseUsername
	rdsSecretPayload.MasterPassword = model.NewRandomPassword(40)
	err = rdsSecretPayload.Validate()
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(&rdsSecretPayload)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal secrets manager payload")
	}

	_, err = a.Service().secretsManager.CreateSecret(
		context.TODO(),
		&secretsmanager.CreateSecretInput{
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

func (a *Client) SecretsManagerCreateSecret(secretName, description string, secretBytes []byte, logger log.FieldLogger) error {
	_, err := a.Service().secretsManager.CreateSecret(
		context.TODO(),
		&secretsmanager.CreateSecretInput{
			Name:         aws.String(secretName),
			Description:  aws.String(description),
			SecretBinary: secretBytes,
		})
	if err != nil {
		return errors.Wrap(err, "unable to create secrets manager secret")
	}

	logger.WithField("secret-name", secretName).Debug("AWS secret created")

	return nil
}

func (a *Client) SecretsManagerUpdateSecret(secretName string, secretBytes []byte, logger log.FieldLogger) error {
	_, err := a.Service().secretsManager.UpdateSecret(
		context.TODO(),
		&secretsmanager.UpdateSecretInput{
			SecretId:     &secretName,
			SecretBinary: secretBytes,
		})
	if err != nil {
		return errors.Wrap(err, "unable to update secrets manager secret")
	}

	logger.WithField("secret-name", secretName).Debug("AWS secret updated")

	return nil
}

// secretsManagerGetIAMAccessKey returns the AccessKey for an IAM account.
func (a *Client) secretsManagerGetIAMAccessKey(awsID string) (*IAMAccessKey, error) {
	return a.secretsManagerGetIAMAccessKeyFromSecretName(IAMSecretName(awsID))
}

// secretsManagerGetIAMAccessKeyFromSecretName attempts to parse a secret into
// and IAMAccessKey.
func (a *Client) secretsManagerGetIAMAccessKeyFromSecretName(secretName string) (*IAMAccessKey, error) {
	result, err := a.Service().secretsManager.GetSecretValue(
		context.TODO(),
		&secretsmanager.GetSecretValueInput{
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

func (a *Client) secretsManagerGetRDSSecret(secretName string) (*RDSSecret, error) {
	result, err := a.Service().secretsManager.GetSecretValue(
		context.TODO(),
		&secretsmanager.GetSecretValueInput{
			SecretId: aws.String(secretName),
		})
	if err != nil {
		return nil, errors.Wrap(err, "unable to get secrets manager secret")
	}

	var rdsSecret *RDSSecret
	err = json.Unmarshal([]byte(*result.SecretString), &rdsSecret)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal secrets manager secret payload")
	}

	err = rdsSecret.Validate()
	if err != nil {
		return nil, err
	}

	return rdsSecret, nil
}

func (a *Client) secretsManagerEnsureIAMAccessKeySecretDeleted(awsID string, logger log.FieldLogger) error {
	return a.secretsManagerEnsureSecretDeleted(IAMSecretName(awsID), false, logger)
}

func (a *Client) secretsManagerEnsureRDSSecretDeleted(awsID string, logger log.FieldLogger) error {
	return a.secretsManagerEnsureSecretDeleted(RDSSecretName(awsID), false, logger)
}

func (a *Client) secretsManagerEnsureSecretDeleted(secretName string, force bool, logger log.FieldLogger) error {
	response, err := a.Service().secretsManager.DeleteSecret(
		context.TODO(),
		&secretsmanager.DeleteSecretInput{
			SecretId:                   aws.String(secretName),
			RecoveryWindowInDays:       aws.Int64(defaultSecretManagerDeletionDays),
			ForceDeleteWithoutRecovery: aws.Bool(force),
		})

	if err != nil {
		var awsErr *types.ResourceNotFoundException
		if errors.As(err, &awsErr) {
			logger.WithField("secret-name", secretName).Warn("Secret Manager secret not found; assuming already deleted")
			return nil
		}
		return errors.Wrap(err, "failed to delete secret")
	}

	logger.WithField("secret-name", secretName).Debugf("Secret Manager secret scheduled for deletion on %s", response.DeletionDate)

	return nil
}

func (a *Client) SecretsManagerGetSecretBytes(secretName string) ([]byte, error) {
	result, err := a.Service().secretsManager.GetSecretValue(
		context.TODO(),
		&secretsmanager.GetSecretValueInput{
			SecretId: aws.String(secretName),
		})
	if err != nil {
		return nil, errors.Wrap(err, "unable to get secrets manager secret")
	}

	return []byte(*result.SecretString), nil
}

func (a *Client) SecretsManagerGetSecretAsK8sSecretData(secretName string) (map[string][]byte, error) {
	result, err := a.Service().secretsManager.GetSecretValue(
		context.TODO(),
		&secretsmanager.GetSecretValueInput{
			SecretId: aws.String(secretName),
		})
	if err != nil {
		return nil, errors.Wrap(err, "unable to get secrets manager secret")
	}

	var data map[string][]byte
	err = json.Unmarshal(result.SecretBinary, &data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to convert AWS secret binary data")
	}

	return data, nil
}

func (a *Client) SecretsManagerEnsureSecretDeleted(secretName string, logger log.FieldLogger) error {
	return a.secretsManagerEnsureSecretDeleted(secretName, false, logger)
}
