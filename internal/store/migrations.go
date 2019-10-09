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
	{semver.MustParse("0.2.0"), semver.MustParse("0.3.0"), func(e execer) error {
		_, err := e.Exec(`
			CREATE TABLE Installation (
				ID CHAR(26) PRIMARY KEY,
				OwnerID CHAR(26) NOT NULL,
				Version VARCHAR(32) NOT NULL,
				DNS VARCHAR(2083) NOT NULL,
				Affinity VARCHAR(32) NOT NULL,
				GroupID CHAR(26) NULL,
				State VARCHAR(32) NOT NULL,
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
			CREATE UNIQUE INDEX Installation_DNS_DeleteAt ON Installation (DNS, DeleteAt);
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
			CREATE TABLE ClusterInstallation (
				ID TEXT PRIMARY KEY,
				ClusterID TEXT NOT NULL,
				InstallationID TEXT NOT NULL,
				Namespace TEXT NOT NULL,
				State TEXT NOT NULL,
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
	{semver.MustParse("0.3.0"), semver.MustParse("0.4.0"), func(e execer) error {
		_, err := e.Exec(`
			CREATE TABLE "Group" (
				ID CHAR(26) PRIMARY KEY,
				Name TEXT,
				Description TEXT,
				Version TEXT,
				CreateAt BIGINT NOT NULL,
				DeleteAt BIGINT NOT NULL
			);
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
			CREATE UNIQUE INDEX Group_Name_DeleteAt ON "Group" (Name, DeleteAt);
		`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.4.0"), semver.MustParse("0.5.0"), func(e execer) error {
		// Switch various columns to TEXT for Postgres. This isn't actually necessary for
		// SQLite, and it's harder to change types there anyway.
		if e.DriverName() == driverPostgres {
			_, err := e.Exec(`ALTER TABLE System ALTER COLUMN Key TYPE TEXT;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE System ALTER COLUMN Value TYPE TEXT;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE Cluster ALTER COLUMN ID TYPE TEXT;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE Cluster ALTER COLUMN Provider TYPE TEXT;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE Cluster ALTER COLUMN Provisioner TYPE TEXT;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE Cluster ALTER COLUMN Size TYPE TEXT;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE Cluster ALTER COLUMN State TYPE TEXT;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE Cluster ALTER COLUMN LockAcquiredBy TYPE TEXT;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE Installation ALTER COLUMN ID TYPE TEXT;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE Installation ALTER COLUMN OwnerId TYPE TEXT;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE Installation ALTER COLUMN Version TYPE TEXT;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE Installation ALTER COLUMN DNS TYPE TEXT;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE Installation ALTER COLUMN Affinity TYPE TEXT;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE Installation ALTER COLUMN GroupID TYPE TEXT;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE Installation ALTER COLUMN State TYPE TEXT;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE Installation ALTER COLUMN LockAcquiredBy TYPE TEXT;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE ClusterInstallation ALTER COLUMN LockAcquiredBy TYPE TEXT;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE "Group" ALTER COLUMN ID TYPE TEXT;`)
			if err != nil {
				return err
			}
		}

		return nil
	}},
	{semver.MustParse("0.5.0"), semver.MustParse("0.6.0"), func(e execer) error {
		// Add the new Size column. SQLite will need to re-create manually.
		if e.DriverName() == driverPostgres {
			_, err := e.Exec(`ALTER TABLE Installation ADD COLUMN Size TEXT DEFAULT '100users';`)
			if err != nil {
				return err
			}
		} else if e.DriverName() == driverSqlite {
			_, err := e.Exec(`ALTER TABLE Installation RENAME TO InstallationTemp;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`
					CREATE TABLE Installation (
						ID TEXT PRIMARY KEY,
						OwnerID TEXT NOT NULL,
						Version TEXT NOT NULL,
						DNS TEXT NOT NULL,
						Size TEXT NOT NULL,
						Affinity TEXT NOT NULL,
						GroupID TEXT NULL,
						State TEXT NOT NULL,
						CreateAt BIGINT NOT NULL,
						DeleteAt BIGINT NOT NULL,
						LockAcquiredBy TEXT NULL,
						LockAcquiredAt BIGINT NOT NULL
					);
				`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`
					INSERT INTO Installation
					SELECT
						ID,
						OwnerID,
						Version,
						DNS,
						"100users",
						Affinity,
						GroupID,
						State,
						CreateAt,
						DeleteAt,
						LockAcquiredBy,
						LockAcquiredAt
					FROM
					InstallationTemp;
				`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`DROP TABLE InstallationTemp;`)
			if err != nil {
				return err
			}
		}

		return nil
	}},
	{semver.MustParse("0.6.0"), semver.MustParse("0.7.0"), func(e execer) error {
		// Add installation license column.
		_, err := e.Exec(`ALTER TABLE Installation ADD COLUMN License TEXT NULL;`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`UPDATE Installation SET License = '';`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.7.0"), semver.MustParse("0.8.0"), func(e execer) error {
		// Add webhook table.
		_, err := e.Exec(`
			CREATE TABLE Webhooks (
				ID TEXT PRIMARY KEY,
				OwnerID TEXT NOT NULL,
				URL TEXT NOT NULL,
				CreateAt BIGINT NOT NULL,
				DeleteAt BIGINT NOT NULL
			);
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
			CREATE UNIQUE INDEX Webhook_URL_DeleteAt ON Webhooks (URL, DeleteAt);
		`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.8.0"), semver.MustParse("0.9.0"), func(e execer) error {
		if e.DriverName() == driverPostgres {
			_, err := e.Exec(`ALTER TABLE Installation ADD COLUMN Database TEXT NULL;`)
			if err != nil {
				return err
			}
			_, err = e.Exec(`ALTER TABLE Installation ADD COLUMN Filestore TEXT NULL;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`UPDATE Installation SET Database = 'mysql-operator';`)
			if err != nil {
				return err
			}
			_, err = e.Exec(`UPDATE Installation SET Filestore = 'minio-operator';`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE Installation ALTER COLUMN Database SET NOT NULL; `)
			if err != nil {
				return err
			}
			_, err = e.Exec(`ALTER TABLE Installation ALTER COLUMN Filestore SET NOT NULL;`)
			if err != nil {
				return err
			}
		} else if e.DriverName() == driverSqlite {
			_, err := e.Exec(`ALTER TABLE Installation RENAME TO InstallationTemp;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`
					CREATE TABLE Installation (
						ID TEXT PRIMARY KEY,
						OwnerID TEXT NOT NULL,
						Version TEXT NOT NULL,
						DNS TEXT NOT NULL,
						Database TEXT NOT NULL,
						Filestore TEXT NOT NULL,
						License TEXT NULL,
						Size TEXT NOT NULL,
						Affinity TEXT NOT NULL,
						GroupID TEXT NULL,
						State TEXT NOT NULL,
						CreateAt BIGINT NOT NULL,
						DeleteAt BIGINT NOT NULL,
						LockAcquiredBy TEXT NULL,
						LockAcquiredAt BIGINT NOT NULL
					);
				`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`
					INSERT INTO Installation
					SELECT
						ID,
						OwnerID,
						Version,
						DNS,
						"mysql-operator",
						"minio-operator",
						License,
						Size,
						Affinity,
						GroupID,
						State,
						CreateAt,
						DeleteAt,
						LockAcquiredBy,
						LockAcquiredAt
					FROM
					InstallationTemp;
				`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`DROP TABLE InstallationTemp;`)
			if err != nil {
				return err
			}
		}

		return nil
	}},
}
