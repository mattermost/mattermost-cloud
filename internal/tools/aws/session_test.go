// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestSanitizeParams(t *testing.T) {
	for _, testCase := range []struct {
		description string
		params      interface{}
		expected    string
	}{
		{
			description: "cannot marshal params",
			params:      "param",
			expected:    "<ALL_PARAMETERS_REDACTED>",
		},
		{
			description: "redact not allowed keys",
			params: map[string]interface{}{
				"Name":               "Awesome Name",
				"Description":        "test",
				"MasterUserPassword": "password",
				"SecretString":       map[string]string{"user": "test", "password": "pass"},
				"Tags": []map[string]interface{}{
					{"some_name": "some value"},
					{"tag": "tag"},
				},
			},
			expected: `{"Description":"test","MasterUserPassword":"*****","Name":"Awesome Name","SecretString":"*****","Tags":[{"some_name":"some value"},{"tag":"tag"}]}`,
		},
		{
			description: "redact custom type",
			params: secretsmanager.CreateSecretInput{
				Name:         aws.String("name"),
				SecretBinary: []byte("secret bytes"),
				SecretString: aws.String("super secret"),
			},
			expected: `{"AddReplicaRegions":null,"ClientRequestToken":null,"Description":null,"ForceOverwriteReplicaSecret":null,"KmsKeyId":null,"Name":"name","SecretBinary":"*****","SecretString":"*****","Tags":null}`,
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			sanitized := sanitizeParams(testCase.params, logrus.New())
			assert.Equal(t, testCase.expected, sanitized)
		})
	}
}
