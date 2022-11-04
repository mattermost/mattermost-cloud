// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package awsv2

import (
	"fmt"

	"emperror.dev/errors"
	"github.com/mattermost/mattermost-cloud/model"
)

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
