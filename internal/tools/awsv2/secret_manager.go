// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package awsv2

import (
	"context"
	"encoding/json"
	"fmt"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

func (a *Client) secretsManagerEnsureRDSSecretCreated(awsID string, logger log.FieldLogger) (interface{}, error) {
	ctx := context.TODO()

	secretName := formatRDSResource(awsID)
	secret := &RDSSecret{}

	// Check if we already have an RDS secret for this installation.
	result, err := a.aws.secretsManager.GetSecretValue(
		ctx,
		&secretsmanager.GetSecretValueInput{
			SecretId: aws.String(secretName),
		},
	)
	if err == nil {
		logger.WithField("secret-name", secretName).Debug("AWS RDS secret already created")

		err = json.Unmarshal([]byte(*result.SecretString), &secret)
		if err != nil {
			return nil, errors.Wrap(err, "unable to marshal secrets manager payload")
		}

		err := secret.Validate()
		if err != nil {
			return nil, err
		}

		return secret, nil
	}

	// There is no existing secret, so we will create a new one with a strong
	// random username and password.
	secret.MasterUsername = model.DefaultMattermostDatabaseUsername
	secret.MasterPassword = model.NewRandomPassword(model.DefaultPasswordLength)
	err = secret.Validate()
	if err != nil {
		return nil, err
	}

	b, err := json.Marshal(&secret)
	if err != nil {
		return nil, errors.Wrap(err, "unable to marshal secrets manager payload")
	}

	_, err = a.aws.secretsManager.CreateSecret(
		ctx,
		&secretsmanager.CreateSecretInput{
			Name:         aws.String(secretName),
			Description:  aws.String(fmt.Sprintf("RDS configuration for %s", awsID)),
			SecretString: aws.String(string(b)),
		})
	if err != nil {
		return nil, errors.Wrap(err, "unable to create secrets manager secret")
	}

	logger.WithField("secret-name", secretName).Debug("AWS RDS secret created")

	return secret, nil
}
