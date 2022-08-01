// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"testing"

	"github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestParseProvisionerSize(t *testing.T) {

	provisionerXL6Replicas := SizeProvisionerXLResources
	provisionerXL6Replicas.App.Replicas = 6

	provisionerXL1Replica := SizeProvisionerXLResources
	provisionerXL1Replica.App.Replicas = 1

	for _, testCase := range []struct {
		description  string
		size         string
		expectedSize v1alpha1.ClusterInstallationSize
		error        string
	}{
		{
			description:  "correct size, no replicas",
			size:         "provisionerXL",
			expectedSize: SizeProvisionerXLResources,
		},
		{
			description:  "correct size, custom replicas (6)",
			size:         "provisionerXL-6",
			expectedSize: provisionerXL6Replicas,
		},
		{
			description: "do not allow more than 2 segments",
			size:        "provisionerXL-1-6",
			error:       "expected at most 2 size segments",
		},
		{
			description: "do not allow dash without replicas",
			size:        "provisionerXL-",
			error:       "replicas segment cannot be empty",
		},
		{
			description: "unknown size",
			size:        "provisionerS",
			error:       "unrecognized installation size",
		},
		{
			description: "invalid repicas value",
			size:        "provisionerXL-two",
			error:       "failed to parse number of replicas from custom provisioner size",
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			resSize, err := ParseProvisionerSize(testCase.size)
			assert.Equal(t, testCase.expectedSize, resSize)
			if testCase.error != "" {
				assert.ErrorContains(t, err, testCase.error)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
