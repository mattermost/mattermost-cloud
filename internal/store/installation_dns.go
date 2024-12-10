// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"
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

// AddInstallationDomain adds Installation domain name to the database.
func (sqlStore *SQLStore) AddInstallationDomain(installation *model.Installation, dnsRecord *model.InstallationDNS) error {
	return sqlStore.createInstallationDNS(sqlStore.db, installation.ID, dnsRecord)
}

type DomainNameInUseError struct {
	DomainName string
}

func (e *DomainNameInUseError) Error() string {
	return fmt.Sprintf("domain name %s already in use", e.DomainName)
}

// isUniqueConstraintViolation checks if the error is a unique constraint violation.
func isUniqueConstraintViolation(err error) bool {
	if pgErr, ok := err.(*pq.Error); ok && pgErr.Code == "23505" {
		return true
	}
	return false
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
		if isUniqueConstraintViolation(err) {
			return &DomainNameInUseError{DomainName: dnsRecord.DomainName}
		}

		return errors.Wrap(err, "failed to create installation DNS record")
	}

	return nil
}

// GetDNSRecordsForInstallation lists DNS Records for specified installation.
func (sqlStore *SQLStore) GetDNSRecordsForInstallation(installationID string) ([]*model.InstallationDNS, error) {
	return sqlStore.getDNSRecordsForInstallations(sqlStore.db, installationID)
}

// GetDNSRecordsForInstallations lists DNS Records for specified installations.
func (sqlStore *SQLStore) GetDNSRecordsForInstallations(installationIDs []string) ([]*model.InstallationDNS, error) {
	return sqlStore.getDNSRecordsForInstallations(sqlStore.db, installationIDs...)
}

func (sqlStore *SQLStore) getDNSRecordsForInstallations(db queryer, installationIDs ...string) ([]*model.InstallationDNS, error) {
	query := sq.Select(installationDNSColumns...).From(installationDNSTable).
		Where(sq.Eq{"InstallationID": installationIDs}).
		Where("DeleteAt = 0").
		OrderBy("CreateAt ASC")

	var records []*model.InstallationDNS
	err := sqlStore.selectBuilder(db, &records, query)
	if err != nil {
		return nil, errors.Wrap(err, "failed to list DNS records for installation")
	}

	return records, nil
}

// SwitchPrimaryInstallationDomain sets given domain as primary reducing all others to non-primary.
func (sqlStore *SQLStore) SwitchPrimaryInstallationDomain(installationID string, installationDNSID string) error {
	tx, err := sqlStore.beginTransaction(sqlStore.db)
	if err != nil {
		return errors.Wrap(err, "failed to start transaction")
	}
	defer tx.RollbackUnlessCommitted()

	query := sq.Update(installationDNSTable).
		Set("IsPrimary", "false").
		Where("installationID = ?", installationID)

	_, err = sqlStore.execBuilder(tx, query)
	if err != nil {
		return errors.Wrap(err, "failed to unset primary DNS for installation")
	}

	query = sq.Update(installationDNSTable).
		Set("IsPrimary", "true").
		Where("ID = ?", installationDNSID)

	result, err := sqlStore.execBuilder(tx, query)
	if err != nil {
		return errors.Wrap(err, "failed to switch primary DNS")
	}
	// Perform validation that exactly one row was updated just in case.
	count, err := result.RowsAffected()
	if err != nil {
		return errors.Wrap(err, "failed to count rows affected on IsPrimary=true update")
	}
	if count != 1 {
		return errors.Errorf("expected exactly 1 row update, but got %d", count)
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
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
