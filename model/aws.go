// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"context"
)

// AWS defines the interface required to operate between packages
type AWS interface {
	GetCertificateByTag(ctx context.Context, key, value string) (*Certificate, error)

	CreateDatabase(dbType DatabaseType, installationID string) (Database, error)
	// ProvisionDatabase() error
}
