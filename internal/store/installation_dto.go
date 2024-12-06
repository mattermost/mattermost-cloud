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

	clusterInstallations, err := sqlStore.GetClusterInstallationsForInstallation(id)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster installations for installation")
	}

	clusterIDs := make([]*string, 0, len(clusterInstallations))
	for _, clusterInstallation := range clusterInstallations {
		clusterIDs = append(clusterIDs, &clusterInstallation.ClusterID)
	}

	return installation.ToDTO(annotations, dnsRecords, clusterIDs), nil
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

	installationIDs := make([]string, 0, len(installations))
	for _, inst := range installations {
		installationIDs = append(installationIDs, inst.ID)
	}
	dnsRecords, err := sqlStore.GetDNSRecordsForInstallations(installationIDs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch DNS records for installations")
	}
	mapping := make(map[string][]*model.InstallationDNS, len(installations))
	for _, record := range dnsRecords {
		mapping[record.InstallationID] = append(mapping[record.InstallationID], record)
	}

	clusterInstallations, err := sqlStore.GetClusterInstallationsForInstallations(installationIDs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get cluster installations for installations")
	}

	clusterMapping := make(map[string][]*string, len(installations))
	for _, clusterInstallation := range clusterInstallations {
		clusterMapping[clusterInstallation.InstallationID] = append(clusterMapping[clusterInstallation.InstallationID], &clusterInstallation.ClusterID)
	}

	dtos := make([]*model.InstallationDTO, 0, len(installations))
	for _, inst := range installations {
		dtos = append(dtos, inst.ToDTO(annotations[inst.ID], mapping[inst.ID], clusterMapping[inst.ID]))
	}

	return dtos, nil
}
