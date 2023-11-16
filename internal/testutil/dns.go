// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package testutil

import "github.com/mattermost/mattermost-cloud/model"

// DNSForInstallation creates slice of DNSRecords for ease of use in tests.
func DNSForInstallation(dns string) []*model.InstallationDNS {
	return []*model.InstallationDNS{
		{
			DomainName: dns,
			IsPrimary:  true,
			CreateAt:   model.GetMillis(),
		},
	}
}
