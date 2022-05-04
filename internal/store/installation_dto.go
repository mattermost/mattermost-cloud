// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

// GetInstallationDTO fetches the given installation by id with data from connected tables.
func (sqlStore *SQLStore) GetInstallationDTO(id string, includeGroupConfig, includeGroupConfigOverrides bool) (*model.InstallationDTO, error) {
	installation, err := sqlStore.GetInstallation(id, includeGroupConfig, includeGroupConfigOverrides)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installation")
	}
	if installation == nil {
		return nil, nil
	}

	annotations, err := sqlStore.GetAnnotationsForInstallation(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get annotations for installation")
	}

	dnsRecords, err := sqlStore.GetDNSRecordsForInstallation(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch DNS records for installation")
	}

	return installation.ToDTO(annotations, dnsRecords), nil
}

// GetInstallationDTOs fetches the given page of installation with data from connected tables. The first page is 0.
func (sqlStore *SQLStore) GetInstallationDTOs(filter *model.InstallationFilter, includeGroupConfig, includeGroupConfigOverrides bool) ([]*model.InstallationDTO, error) {
	installations, err := sqlStore.GetInstallations(filter, includeGroupConfig, includeGroupConfigOverrides)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get installations")
	}

	annotations, err := sqlStore.GetAnnotationsForInstallations(filter)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get annotations for installations")
	}

	dtos := make([]*model.InstallationDTO, 0, len(installations))
	for _, inst := range installations {

		// TODO: we could try to make it a single query as for annotations.
		dnsRecords, err := sqlStore.GetDNSRecordsForInstallation(inst.ID)
		if err != nil {
			return nil, errors.Wrap(err, "failed to fetch DNS records for installations")
		}

		dtos = append(dtos, inst.ToDTO(annotations[inst.ID], dnsRecords))
	}

	return dtos, nil
}
