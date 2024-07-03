// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/pkg/errors"
)

// SecretsManagerValidateExternalClusterSecret validates a secret for use in
// an externally-managed cluster.
func (a *Client) SecretsManagerValidateExternalClusterSecret(name string) error {
	_, err := a.Service().secretsManager.GetSecretValue(
		context.TODO(),
		&secretsmanager.GetSecretValueInput{
			SecretId: &name,
		})
	if err != nil {
		return errors.Wrap(err, "failed to get secret value for cluster")
	}

	return nil
}
