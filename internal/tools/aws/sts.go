// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"github.com/aws/aws-sdk-go/service/sts"
)

// GetAccountID gets the current AWS Account ID
func (a *Client) GetAccountID() (string, error) {
	callerIdentityOutput, err := a.Service().sts.GetCallerIdentity(&sts.GetCallerIdentityInput{})
	if err != nil {
		return "", err
	}
	return *callerIdentityOutput.Account, nil
}
