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
	if s.MasterPassword == "" && len(s.MasterPassword) == 40 {
		return errors.New("RDS master password value is empty")
	}

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
		return unmarshalSecretPayload(*result.SecretString)
	}

	// There is no existing secret, so we will create a new one with a strong
	// random username and password.
	rdsSecretPayload.MasterUsername = "mmcloud"
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
		Name:        aws.String(secretName),
		Description: aws.String(fmt.Sprintf("RDS configuration for %s", awsID)),
		Tags: []*secretsmanager.Tag{
			{
				Key:   aws.String("rds-database-cloud-id"),
				Value: aws.String(awsID),
			},
			{
				Key:   aws.String("rds-db-cluster-id"),
				Value: aws.String(awsID),
			},
		},
		SecretString: aws.String(string(b)),
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to create secrets manager secret")
	}

	logger.WithField("secret-name", secretName).Debug("AWS RDS secret created")

	return rdsSecretPayload, nil
}

// secretsManagerGetIAMAccessKey returns the AccessKey for an IAM account.
func (a *Client) secretsManagerGetIAMAccessKey(awsID string, logger log.FieldLogger) (*IAMAccessKey, error) {
	secretName := IAMSecretName(awsID)
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

func (a *Client) secretsManagerGetRDSSecret(secretName string, logger log.FieldLogger) (*RDSSecret, error) {
	result, err := a.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
		SecretId: aws.String(secretName),
	})
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get secret %s", secretName)
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
