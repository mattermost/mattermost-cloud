// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

const (
	installationDNSTable = "InstallationDNS"
)

var (
	installationDNSColumns = []string{"ID", "DomainName", "InstallationID", "IsPrimary", "CreateAt", "DeleteAt"}
)

// GetInstallationDNS returns InstallationDNS with given id.
func (sqlStore *SQLStore) GetInstallationDNS(id string) (*model.InstallationDNS, error) {
	query := sq.Select(installationDNSColumns...).
		From(installationDNSTable).
		Where("ID = ?", id)

	var installationDNS model.InstallationDNS
	err := sqlStore.getBuilder(sqlStore.db, &installationDNS, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(err, "failed to get installation DNS")
	}
	return &installationDNS, nil
}

func (sqlStore *SQLStore) createInstallationDNS(db execer, installationID string, dnsRecord *model.InstallationDNS) error {
	dnsRecord.ID = model.NewID()
	dnsRecord.InstallationID = installationID
	dnsRecord.CreateAt = model.GetMillis()

	query := sq.Insert(installationDNSTable).SetMap(map[string]interface{}{
		"ID":             dnsRecord.ID,
		"DomainName":     dnsRecord.DomainName,
		"InstallationID": dnsRecord.InstallationID,
		"IsPrimary":      dnsRecord.IsPrimary,
		"CreateAt":       dnsRecord.CreateAt,
		"DeleteAt":       dnsRecord.DeleteAt,
	})

	_, err := sqlStore.execBuilder(db, query)
	if err != nil {
		return errors.Wrap(err, "failed to create installation DNS record")
	}

	return nil
}

// GetDNSRecordsForInstallation lists DNS Records for specified installation.
func (sqlStore *SQLStore) GetDNSRecordsForInstallation(installationID string) ([]*model.InstallationDNS, error) {
	return sqlStore.getDNSRecordsForInstallation(sqlStore.db, installationID)
}

func (sqlStore *SQLStore) getDNSRecordsForInstallation(db queryer, installationID string) ([]*model.InstallationDNS, error) {
	query := sq.Select(installationDNSColumns...).From(installationDNSTable).
		Where("InstallationID = ?", installationID).
		Where("DeleteAt = 0").
		OrderBy("CreateAt ASC")

	var records []*model.InstallationDNS
	err := sqlStore.selectBuilder(db, &records, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list DNS records for installation")
	}

	return records, nil
}

// DeleteInstallationDNS marks Installation DNS record as deleted.
func (sqlStore *SQLStore) DeleteInstallationDNS(installationID, dnsName string) error {
	query := sq.Update(installationDNSTable).Set("DeleteAt", model.GetMillis()).
		Where("InstallationID = ?", installationID).
		Where("DomainName = ?", dnsName).
		Where("DeleteAt = 0")

	_, err := sqlStore.execBuilder(sqlStore.db, query)
	if err != nil {
		return errors.Wrap(err, "failed to delete DNS record for installation")
	}

	return nil
}
