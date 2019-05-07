package store

import (
	"github.com/blang/semver"
)

type migration struct {
	fromVersion   semver.Version
	toVersion     semver.Version
	migrationFunc func(execer) error
}

// migrations defines the set of migrations necessary to advance the database to the latest
// expected version.
//
// Note that the canonical schema is currently obtained by applying all migrations to an empty
// database.
var migrations = []migration{
	{semver.MustParse("0.0.0"), semver.MustParse("0.1.0"), func(e execer) error {
		_, err := e.Exec(`
			CREATE TABLE System (
				Key VARCHAR(64) PRIMARY KEY,
				Value VARCHAR(1024) NULL
			);
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
			CREATE TABLE Cluster (
				ID CHAR(26) PRIMARY KEY,
				Provider VARCHAR(32) NOT NULL,
				Provisioner VARCHAR(32) NOT NULL,
				ProviderMetadata BYTEA NULL,
				ProvisionerMetadata BYTEA NULL,
				AllowInstallations BOOLEAN NOT NULL,
				CreateAt BIGINT NOT NULL,
				DeleteAt BIGINT NOT NULL,
				LockAcquiredBy CHAR(26) NULL,
				LockAcquiredAt BIGINT NOT NULL
			);
		`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.1.0"), semver.MustParse("0.2.0"), func(e execer) error {
		if e.DriverName() == driverPostgres {
			_, err := e.Exec(`ALTER TABLE Cluster ADD COLUMN Size VARCHAR(32) NULL;`)
			if err != nil {
				return err
			}
			_, err = e.Exec(`ALTER TABLE Cluster ADD COLUMN State VARCHAR(32) NULL;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`UPDATE Cluster SET Size = 'SizeAlef500';`)
			if err != nil {
				return err
			}
			_, err = e.Exec(`UPDATE Cluster SET State = 'stable';`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE Cluster ALTER COLUMN Size SET NOT NULL; `)
			if err != nil {
				return err
			}
			_, err = e.Exec(`ALTER TABLE Cluster ALTER COLUMN State SET NOT NULL;`)
			if err != nil {
				return err
			}
		} else if e.DriverName() == driverSqlite {
			_, err := e.Exec(`ALTER TABLE Cluster RENAME TO ClusterTemp;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`
				CREATE TABLE Cluster (
					ID CHAR(26) PRIMARY KEY,
					Provider VARCHAR(32) NOT NULL,
					Provisioner VARCHAR(32) NOT NULL,
					ProviderMetadata BYTEA NULL,
					ProvisionerMetadata BYTEA NULL,
					Size VARCHAR(32) NOT NULL,
					State VARCHAR(32) NOT NULL,
					AllowInstallations BOOLEAN NOT NULL,
					CreateAt BIGINT NOT NULL,
					DeleteAt BIGINT NOT NULL,
					LockAcquiredBy CHAR(26) NULL,
					LockAcquiredAt BIGINT NOT NULL
				);
			`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`
				INSERT INTO Cluster 
				SELECT
					ID,
					Provider,
					Provisioner,
					ProviderMetadata,
					ProvisionerMetadata,
					"SizeAlef500",
					"stable",
					AllowInstallations,
					CreateAt,
					DeleteAt,
					LockAcquiredBy,
					LockAcquiredAt
				FROM
					ClusterTemp;
			`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`DROP TABLE ClusterTemp;`)
			if err != nil {
				return err
			}
		}

		return nil
	}},
}
