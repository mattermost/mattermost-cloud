// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

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
	{semver.MustParse("0.9.0"), semver.MustParse("0.10.0"), func(e execer) error {
		if e.DriverName() == driverPostgres {
			_, err := e.Exec(`UPDATE Cluster SET AllowInstallations = 'TRUE';`)
			if err != nil {
				return err
			}
		} else if e.DriverName() == driverSqlite {
			_, err := e.Exec(`UPDATE Cluster SET AllowInstallations = '1';`)
			if err != nil {
				return err
			}
		}

		return nil
	}},
	{semver.MustParse("0.10.0"), semver.MustParse("0.11.0"), func(e execer) error {
		// Add Cluster Version column.
		// Also convert SQLite char/varchar columns to type TEXT to match PG.
		if e.DriverName() == driverPostgres {
			_, err := e.Exec(`ALTER TABLE Cluster ADD COLUMN Version TEXT NULL;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`UPDATE Cluster SET Version = '0.0.0';`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE Cluster ALTER COLUMN Version SET NOT NULL; `)
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
					ID TEXT PRIMARY KEY,
					Provider TEXT NOT NULL,
					Provisioner TEXT NOT NULL,
					ProviderMetadata BYTEA NULL,
					ProvisionerMetadata BYTEA NULL,
					Version TEXT NOT NULL,
					Size TEXT NOT NULL,
					State TEXT NOT NULL,
					AllowInstallations BOOLEAN NOT NULL,
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
				INSERT INTO Cluster
				SELECT
					ID,
					Provider,
					Provisioner,
					ProviderMetadata,
					ProvisionerMetadata,
					"0.0.0",
					Size,
					State,
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
	{semver.MustParse("0.11.0"), semver.MustParse("0.12.0"), func(e execer) error {
		// Add version columns for all of the utilities.
		_, err := e.Exec(`
				ALTER TABLE Cluster ADD COLUMN UtilityMetadata BYTEA NULL;
		`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.12.0"), semver.MustParse("0.12.1"), func(e execer) error {
		// Assign default values to UtilityMetadata if values do not
		// exist, otherwise, keep existing values
		_, err := e.Exec(`
				UPDATE Cluster
				SET UtilityMetadata = '{}' WHERE UtilityMetadata is NULL;
		 `)
		return err
	}},
	{semver.MustParse("0.12.1"), semver.MustParse("0.13.0"), func(e execer) error {
		// Add Mattermost Env column for installations.
		_, err := e.Exec(`
				ALTER TABLE Installation
				ADD COLUMN MattermostEnvRaw BYTEA NULL;
		`)
		return err
	}},
	{semver.MustParse("0.13.0"), semver.MustParse("0.14.0"), func(e execer) error {
		// Add Mattermost Env column for groups.
		_, err := e.Exec(`
				ALTER TABLE "Group"
				ADD COLUMN MattermostEnvRaw BYTEA NULL;
				`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.14.0"), semver.MustParse("0.15.0"), func(e execer) error {
		// Add Group Sequence column for installations and groups.
		// Add Group MaxRolling column.
		// Add Group lock columns.
		// tl;dr we should get rid of SQLite

		_, err := e.Exec(`ALTER TABLE Installation ADD COLUMN GroupSequence BIGINT NULL;`)
		if err != nil {
			return err
		}

		if e.DriverName() == driverPostgres {
			_, err = e.Exec(`ALTER TABLE "Group" ADD COLUMN Sequence BIGINT NULL;`)
			if err != nil {
				return err
			}
			_, err = e.Exec(`UPDATE "Group" SET Sequence = '0';`)
			if err != nil {
				return err
			}
			_, err = e.Exec(`ALTER TABLE "Group" ALTER COLUMN Sequence SET NOT NULL; `)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE "Group" ADD COLUMN MaxRolling BIGINT NULL;`)
			if err != nil {
				return err
			}
			_, err = e.Exec(`UPDATE "Group" SET MaxRolling = '1';`)
			if err != nil {
				return err
			}
			_, err = e.Exec(`ALTER TABLE "Group" ALTER COLUMN MaxRolling SET NOT NULL; `)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE "Group" ADD COLUMN LockAcquiredBy TEXT NULL;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE "Group" ADD COLUMN LockAcquiredAt BIGINT NULL;`)
			if err != nil {
				return err
			}
			_, err = e.Exec(`UPDATE "Group" SET LockAcquiredAt = '0';`)
			if err != nil {
				return err
			}
			_, err = e.Exec(`ALTER TABLE "Group" ALTER COLUMN LockAcquiredAt SET NOT NULL; `)
			if err != nil {
				return err
			}

		} else if e.DriverName() == driverSqlite {
			_, err := e.Exec(`ALTER TABLE "Group" RENAME TO "GroupTemp";`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`
				CREATE TABLE "Group" (
					ID TEXT PRIMARY KEY,
					Name TEXT,
					Description TEXT,
					Version TEXT,
					MattermostEnvRaw BYTEA NULL,
					MaxRolling BIGINT NOT NULL,
					Sequence BIGINT NOT NULL,
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
					INSERT INTO "Group"
					SELECT
						ID,
						Name,
						Description,
						Version,
						MattermostEnvRaw,
						1,
						0,
						CreateAt,
						DeleteAt,
						NULL,
						0
					FROM
						"GroupTemp";
				`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`DROP TABLE "GroupTemp";`)
			if err != nil {
				return err
			}
		}

		return nil
	}},
	{semver.MustParse("0.15.0"), semver.MustParse("0.16.0"), func(e execer) error {
		if e.DriverName() == driverPostgres {
			// Add Mattermost Image for installations.
			_, err := e.Exec(`
				ALTER TABLE Installation
				ADD COLUMN Image TEXT NULL;
				`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`
				UPDATE Installation
				SET Image = 'mattermost/mattermost-enterprise-edition';
		 		`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE Installation ALTER COLUMN Image SET NOT NULL;`)
			if err != nil {
				return err
			}

			// Add Mattermost Image for groups.
			_, err = e.Exec(`
				ALTER TABLE "Group"
				ADD COLUMN Image TEXT NULL;
				`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`
				UPDATE "Group"
				SET Image = 'mattermost/mattermost-enterprise-edition';
		 		`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`ALTER TABLE "Group" ALTER COLUMN Image SET NOT NULL;`)
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
						Image TEXT NOT NULL,
						DNS TEXT NOT NULL,
						Database TEXT NOT NULL,
						Filestore TEXT NOT NULL,
						License TEXT NULL,
						Size TEXT NOT NULL,
						MattermostEnvRaw BYTEA NULL,
						Affinity TEXT NOT NULL,
						GroupSequence BIGINT NULL,
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
						"mattermost/mattermost-enterprise-edition",
						DNS,
						Database,
						Filestore,
						License,
						Size,
						MattermostEnvRaw,
						Affinity,
						GroupSequence,
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

			_, err = e.Exec(`ALTER TABLE "Group" RENAME TO "GroupTemp";`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`
				CREATE TABLE "Group" (
					ID TEXT PRIMARY KEY,
					Name TEXT,
					Description TEXT,
					Version TEXT,
					Image TEXT,
					MattermostEnvRaw BYTEA NULL,
					MaxRolling BIGINT NOT NULL,
					Sequence BIGINT NOT NULL,
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
				INSERT INTO "Group"
					SELECT
						ID,
						Name,
						Description,
						Version,
						"mattermost/mattermost-enterprise-edition",
						MattermostEnvRaw,
						MaxRolling,
						Sequence,
						CreateAt,
						DeleteAt,
						LockAcquiredBy,
						LockAcquiredAt
					FROM
						"GroupTemp";
					`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`DROP TABLE GroupTemp;`)
			if err != nil {
				return err
			}
		}

		return nil
	}},
	{semver.MustParse("0.16.0"), semver.MustParse("0.17.0"), func(e execer) error {
		_, err := e.Exec(`
				CREATE TABLE MultitenantDatabase (
					ID TEXT PRIMARY KEY,
					RawInstallationIDs BYTEA NOT NULL,
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
	{semver.MustParse("0.17.0"), semver.MustParse("0.18.0"), func(e execer) error {
		_, err := e.Exec(`
			ALTER TABLE MultitenantDatabase
			ADD COLUMN VpcID TEXT NULL;
		`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.18.0"), semver.MustParse("0.19.0"), func(e execer) error {
		_, err := e.Exec(`ALTER TABLE Cluster RENAME TO ClusterTemp;`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
			CREATE TABLE Cluster (
				ID TEXT PRIMARY KEY,
				Provider TEXT NOT NULL,
				Provisioner TEXT NOT NULL,
				ProviderMetadataRaw BYTEA NULL,
				ProvisionerMetadataRaw BYTEA NULL,
				UtilityMetadataRaw BYTEA NULL,
				Version TEXT NOT NULL,
				Size TEXT NOT NULL,
				State TEXT NOT NULL,
				AllowInstallations BOOLEAN NOT NULL,
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
			INSERT INTO Cluster
			SELECT
				ID,
				Provider,
				Provisioner,
				ProviderMetadata,
				ProvisionerMetadata,
				UtilityMetadata,
				Version,
				Size,
				State,
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

		return nil
	}},
	{semver.MustParse("0.19.0"), semver.MustParse("0.20.0"), func(e execer) error {
		// Changes:
		// 1. Add APISecurityLock to cluster, installation, cluster installation
		//    and group resources.
		// 2. Remove deprecated cluster fields.

		_, err := e.Exec(`ALTER TABLE Cluster RENAME TO ClusterTemp;`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
			CREATE TABLE Cluster (
				ID TEXT PRIMARY KEY,
				Provider TEXT NOT NULL,
				Provisioner TEXT NOT NULL,
				ProviderMetadataRaw BYTEA NULL,
				ProvisionerMetadataRaw BYTEA NULL,
				UtilityMetadataRaw BYTEA NULL,
				State TEXT NOT NULL,
				AllowInstallations BOOLEAN NOT NULL,
				CreateAt BIGINT NOT NULL,
				DeleteAt BIGINT NOT NULL,
				APISecurityLock BOOLEAN NOT NULL,
				LockAcquiredBy TEXT NULL,
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
				ProviderMetadataRaw,
				ProvisionerMetadataRaw,
				UtilityMetadataRaw,
				State,
				AllowInstallations,
				CreateAt,
				DeleteAt,
				'false',
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

		_, err = e.Exec(`ALTER TABLE Installation ADD COLUMN APISecurityLock BOOLEAN NOT NULL DEFAULT 'false';`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`ALTER TABLE ClusterInstallation ADD COLUMN APISecurityLock BOOLEAN NOT NULL DEFAULT 'false';`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`ALTER TABLE "Group" ADD COLUMN APISecurityLock BOOLEAN NOT NULL DEFAULT 'false';`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.20.0"), semver.MustParse("0.21.0"), func(e execer) error {
		// Changes:
		// 1. Add DatabaseType column.
		// 2. Rename RawInstallationIDs to InstallationsRaw.
		// 3. Set VpcID to NOT NULL.

		_, err := e.Exec(`ALTER TABLE MultitenantDatabase RENAME TO MultitenantDatabaseTemp;`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
			CREATE TABLE MultitenantDatabase (
				ID TEXT PRIMARY KEY,
				VpcID TEXT NOT NULL,
				DatabaseType TEXT NOT NULL,
				InstallationsRaw BYTEA NOT NULL,
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
		INSERT INTO MultitenantDatabase
		SELECT
			ID,
			VpcID,
			'mysql',
			RawInstallationIDs,
			CreateAt,
			DeleteAt,
			LockAcquiredBy,
			LockAcquiredAt
		FROM
		MultitenantDatabaseTemp;
	`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`DROP TABLE MultitenantDatabaseTemp;`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.21.0"), semver.MustParse("0.22.0"), func(e execer) error {
		// Changes:
		// 1. Add Annotation table.
		// 2. Add ClusterAnnotation table.
		// 3. Add InstallationAnnotation table.
		// 4. Add constraints to ensure Annotation to Cluster mappings and Annotation to Installation mappings are unique.

		_, err := e.Exec(`
			CREATE TABLE Annotation (
				ID TEXT PRIMARY KEY,
				Name TEXT NOT NULL UNIQUE
			);
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
			CREATE TABLE ClusterAnnotation (
				ID TEXT PRIMARY KEY,
				ClusterID TEXT NOT NULL,
				AnnotationID TEXT NOT NULL
			);
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
			CREATE UNIQUE INDEX ClusterAnnotation_ClusterID_AnnotationID ON ClusterAnnotation (ClusterID, AnnotationID);
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
			CREATE TABLE InstallationAnnotation (
				ID TEXT PRIMARY KEY,
				InstallationID TEXT NOT NULL,
				AnnotationID TEXT NOT NULL
			);
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
			CREATE UNIQUE INDEX InstallationAnnotation_InstallationID_AnnotationID ON InstallationAnnotation (InstallationID, AnnotationID);
		`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.22.0"), semver.MustParse("0.23.0"), func(e execer) error {
		// Add SingleTenantDatabaseConfigRaw column for installations.
		_, err := e.Exec(`
				ALTER TABLE Installation
				ADD COLUMN SingleTenantDatabaseConfigRaw BYTEA NULL;
				`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.23.0"), semver.MustParse("0.24.0"), func(e execer) error {
		// Add CRVersion column for installations.
		_, err := e.Exec(`
				ALTER TABLE Installation
				ADD COLUMN CRVersion TEXT NOT NULL DEFAULT 'mattermost.com/v1alpha1';
				`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.24.0"), semver.MustParse("0.25.0"), func(e execer) error {
		// Add InstallationBackup table.
		_, err := e.Exec(`
			CREATE TABLE InstallationBackup (
				ID TEXT PRIMARY KEY,
				InstallationID TEXT NOT NULL,
				ClusterInstallationID TEXT NOT NULL,
				DataResidenceRaw BYTEA NULL,
				State TEXT NOT NULL,
				RequestAt BIGINT NOT NULL,
				StartAt BIGINT NOT NULL,
				DeleteAt BIGINT NOT NULL,
				APISecurityLock BOOLEAN NOT NULL, 
				LockAcquiredBy TEXT NULL,
				LockAcquiredAt BIGINT NOT NULL
			);
			`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.25.0"), semver.MustParse("0.26.0"), func(e execer) error {
		// Add InstallationDBRestorationOperation table.
		_, err := e.Exec(`
			CREATE TABLE InstallationDBRestorationOperation (
				ID TEXT PRIMARY KEY,
				InstallationID TEXT NOT NULL,
				BackupID TEXT NOT NULL,
				RequestAt BIGINT NOT NULL,
				State TEXT NOT NULL,
				TargetInstallationState TEXT NOT NULL,
				ClusterInstallationID TEXT NOT NULL,
				CompleteAt BIGINT NOT NULL,
				DeleteAt BIGINT NOT NULL,
				LockAcquiredBy TEXT NULL,
				LockAcquiredAt BIGINT NOT NULL
			);
		`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.26.0"), semver.MustParse("0.27.0"), func(e execer) error {
		// 1. Add InstallationDBMigrationOperation table.
		// 2. Add column MigratedInstallationsRaw to MultitenantDatabase table.
		_, err := e.Exec(`
			CREATE TABLE InstallationDBMigrationOperation (
				ID TEXT PRIMARY KEY,
				InstallationID TEXT NOT NULL,
				RequestAt BIGINT NOT NULL,
				State TEXT NOT NULL,
				SourceDatabase TEXT NOT NULL,
				DestinationDatabase TEXT NOT NULL,
				SourceMultiTenantRaw BYTEA NULL,
				DestinationMultiTenantRaw BYTEA NULL,
				BackupID TEXT NOT NULL,
				InstallationDBRestorationOperationID TEXT NOT NULL,
				CompleteAt BIGINT NOT NULL,
				DeleteAt BIGINT NOT NULL,
				LockAcquiredBy TEXT NULL,
				LockAcquiredAt BIGINT NOT NULL
			);
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`ALTER TABLE MultitenantDatabase RENAME TO MultitenantDatabaseTemp;`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
			CREATE TABLE MultitenantDatabase (
				ID TEXT PRIMARY KEY,
				VpcID TEXT NOT NULL,
				DatabaseType TEXT NOT NULL,
				InstallationsRaw BYTEA NOT NULL,
				MigratedInstallationsRaw BYTEA NOT NULL,
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
		INSERT INTO MultitenantDatabase
		SELECT
			ID,
			VpcID,
			DatabaseType,
			InstallationsRaw,
			'[]',
			CreateAt,
			DeleteAt,
			LockAcquiredBy,
			LockAcquiredAt
		FROM
		MultitenantDatabaseTemp;
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`DROP TABLE MultitenantDatabaseTemp;`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.27.0"), semver.MustParse("0.28.0"), func(e execer) error {
		// Add column BackedUpDatabaseType to InstallationBackup table

		_, err := e.Exec(`ALTER TABLE InstallationBackup
				ADD COLUMN BackedUpDatabaseType TEXT NOT NULL DEFAULT 'aws-multitenant-rds-postgres';`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.28.0"), semver.MustParse("0.29.0"), func(e execer) error {
		_, err := e.Exec(`
				ALTER TABLE MultitenantDatabase
				ADD COLUMN State TEXT NOT NULL DEFAULT 'stable';
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
				ALTER TABLE MultitenantDatabase
				ADD COLUMN SharedLogicalDatabaseMappingsRaw BYTEA NULL;
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
				ALTER TABLE MultitenantDatabase
				ADD COLUMN MaxInstallationsPerLogicalDatabase BIGINT NOT NULL DEFAULT '0';
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
				ALTER TABLE MultitenantDatabase
				ADD COLUMN WriterEndpoint TEXT NULL;
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
				ALTER TABLE MultitenantDatabase
				ADD COLUMN ReaderEndpoint TEXT NULL;
		`)
		if err != nil {
			return err
		}

		return nil
	}},
}
