// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAddInstallationDNS_Validate(t *testing.T) {

	for _, testCase := range []struct {
		description      string
		isError          bool
		installationName string
		request          *AddDNSRecordRequest
	}{
		{
			description:      "valid DNS",
			isError:          false,
			installationName: "my-installation",
			request:          &AddDNSRecordRequest{DNS: "my-installation.dns.com"},
		},
		{
			description:      "invalid DNS",
			isError:          true,
			installationName: "my-installation",
			request:          &AddDNSRecordRequest{DNS: "my-installation. dns.com"},
		},
		{
			description:      "DNS does not start with installation name",
			isError:          true,
			installationName: "my-installation",
			request:          &AddDNSRecordRequest{DNS: "not-my-installation.dns.com"},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			err := testCase.request.Validate(testCase.installationName)
			if testCase.isError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}

}
