// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"fmt"

	"github.com/pkg/errors"
)

// GenericCredential stores a generic username and password authentication values
type GenericCredential struct {
	Username string
	Password string
}

// Validate validates the credential for the minimum required for the credential to be consider valid and usable
func (c GenericCredential) Validate() error {
	if c.Username == "" {
		return errors.New("RDS master username value is empty")
	}
	if c.Password == "" {
		return errors.New("RDS master password value is empty")
	}
	if len(c.Password) != DefaultPasswordLength {
		return errors.New(fmt.Sprintf("RDS master password length should be equal to %d", DefaultPasswordLength))
	}

	return nil
}
