// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"encoding/json"
	"fmt"

	"github.com/blang/semver"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
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
		// Add IsStale status column for ClusterInstallation.
		_, err := e.Exec(`ALTER TABLE ClusterInstallation ADD COLUMN IsActive BOOLEAN NOT NULL DEFAULT 'true';`)
		if err != nil {
			return err
		}
		if e.DriverName() == driverSqlite {
			_, err := e.Exec(`UPDATE ClusterInstallation SET IsActive = '1';`)
			if err != nil {
				return err
			}
		}
		return nil
	}},
	{semver.MustParse("0.29.0"), semver.MustParse("0.30.0"), func(e execer) error {
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
				ADD COLUMN WriterEndpoint TEXT NOT NULL DEFAULT '';
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
				ALTER TABLE MultitenantDatabase
				ADD COLUMN ReaderEndpoint TEXT NOT NULL DEFAULT '';
		`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.30.0"), semver.MustParse("0.31.0"), func(e execer) error {
		_, err := e.Exec(`
				ALTER TABLE MultitenantDatabase
				ADD COLUMN NewID TEXT NOT NULL DEFAULT '';
		`)
		if err != nil {
			return err
		}

		multitenantDatabaseRows, err := e.Query(`SELECT ID FROM MultitenantDatabase;`)
		if err != nil {
			return err
		}
		defer multitenantDatabaseRows.Close()

		var id string
		var ids []string
		for multitenantDatabaseRows.Next() {
			err = multitenantDatabaseRows.Scan(&id)
			if err != nil {
				return err
			}
			ids = append(ids, id)
		}
		err = multitenantDatabaseRows.Err()
		if err != nil {
			return err
		}

		for _, id := range ids {
			_, err = e.Exec(`UPDATE MultitenantDatabase SET NewID = $1 WHERE ID = $2;`, model.NewID(), id)
			if err != nil {
				return err
			}
		}

		_, err = e.Exec(`ALTER TABLE MultitenantDatabase RENAME TO MultitenantDatabaseBackup;`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
			CREATE TABLE MultitenantDatabase (
				ID TEXT PRIMARY KEY,
				RdsClusterID TEXT NOT NULL,
				VpcID TEXT NOT NULL,
				DatabaseType TEXT NOT NULL,
				State TEXT NOT NULL,
				WriterEndpoint TEXT NOT NULL,
				ReaderEndpoint TEXT NOT NULL,
				InstallationsRaw BYTEA NOT NULL,
				MigratedInstallationsRaw BYTEA NOT NULL,
				SharedLogicalDatabaseMappingsRaw BYTEA NULL,
				MaxInstallationsPerLogicalDatabase BIGINT NOT NULL,
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
			CREATE UNIQUE INDEX MultitenantDatabase_RdsClusterID_DeleteAt ON MultitenantDatabase (RdsClusterID, DeleteAt);
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
		INSERT INTO MultitenantDatabase
		SELECT
			NewID,
			ID,
			VpcID,
			DatabaseType,
			State,
			WriterEndpoint,
			ReaderEndpoint,
			InstallationsRaw,
			MigratedInstallationsRaw,
			SharedLogicalDatabaseMappingsRaw,
			MaxInstallationsPerLogicalDatabase,
			CreateAt,
			DeleteAt,
			LockAcquiredBy,
			LockAcquiredAt
		FROM
		MultitenantDatabaseBackup;
		`)
		if err != nil {
			return err
		}

		// The MultitenantDatabaseBackup table will be left for now just in case
		// and will be cleaned up in a future migration.

		return nil
	}},
	{semver.MustParse("0.31.0"), semver.MustParse("0.32.0"), func(e execer) error {
		_, err := e.Exec(`ALTER TABLE MultitenantDatabase RENAME TO MultitenantDatabaseBackup2;`)
		if err != nil {
			return err
		}

		multitenantDatabaseRows, err := e.Query(`SELECT ID, SharedLogicalDatabaseMappingsRaw FROM MultitenantDatabaseBackup2;`)
		if err != nil {
			return err
		}
		defer multitenantDatabaseRows.Close()

		sharedDatabaseMapping := make(map[string]map[string][]string)
		for multitenantDatabaseRows.Next() {
			var id string
			var sharedLogicalDatabaseMappingsRaw []byte

			err = multitenantDatabaseRows.Scan(&id, &sharedLogicalDatabaseMappingsRaw)
			if err != nil {
				return err
			}

			var sharedLogicalDatabases map[string][]string
			if sharedLogicalDatabaseMappingsRaw != nil {
				err = json.Unmarshal(sharedLogicalDatabaseMappingsRaw, &sharedLogicalDatabases)
				if err != nil {
					return err
				}
			}
			sharedDatabaseMapping[id] = sharedLogicalDatabases
		}
		err = multitenantDatabaseRows.Err()
		if err != nil {
			return err
		}

		_, err = e.Exec(`
			CREATE TABLE LogicalDatabase (
				ID TEXT PRIMARY KEY,
				MultitenantDatabaseID TEXT NOT NULL,
				Name TEXT NOT NULL,
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
			CREATE TABLE DatabaseSchema (
				ID TEXT PRIMARY KEY,
				LogicalDatabaseID TEXT NOT NULL,
				InstallationID TEXT NOT NULL,
				Name TEXT NOT NULL,
				CreateAt BIGINT NOT NULL,
				DeleteAt BIGINT NOT NULL,
				LockAcquiredBy TEXT NULL,
				LockAcquiredAt BIGINT NOT NULL
			);
		`)
		if err != nil {
			return err
		}

		for multitenantDatabaseID, m := range sharedDatabaseMapping {
			for logicalDatabaseName, installationIDs := range m {
				logicalDatabaseID := model.NewID()
				_, err = e.Exec(`
					INSERT INTO LogicalDatabase VALUES ($1, $2, $3, $4, 0, NULL, 0);`,
					logicalDatabaseID,
					multitenantDatabaseID,
					logicalDatabaseName,
					model.GetMillis(),
				)
				if err != nil {
					return err
				}

				for _, installationID := range installationIDs {
					_, err = e.Exec(`
						INSERT INTO DatabaseSchema VALUES ($1, $2, $3, $4, $5, 0, NULL, 0);`,
						model.NewID(),
						logicalDatabaseID,
						installationID,
						fmt.Sprintf("id_%s", installationID),
						model.GetMillis(),
					)
					if err != nil {
						return err
					}
				}
			}
		}

		_, err = e.Exec(`
			CREATE TABLE MultitenantDatabase (
				ID TEXT PRIMARY KEY,
				RdsClusterID TEXT NOT NULL,
				VpcID TEXT NOT NULL,
				DatabaseType TEXT NOT NULL,
				State TEXT NOT NULL,
				WriterEndpoint TEXT NOT NULL,
				ReaderEndpoint TEXT NOT NULL,
				InstallationsRaw BYTEA NOT NULL,
				MigratedInstallationsRaw BYTEA NOT NULL,
				MaxInstallationsPerLogicalDatabase BIGINT NOT NULL,
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
			DROP INDEX MultitenantDatabase_RdsClusterID_DeleteAt;
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
			CREATE UNIQUE INDEX MultitenantDatabase_RdsClusterID_DeleteAt ON MultitenantDatabase (RdsClusterID, DeleteAt);
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
			INSERT INTO MultitenantDatabase
			SELECT
				ID,
				RdsClusterID,
				VpcID,
				DatabaseType,
				State,
				WriterEndpoint,
				ReaderEndpoint,
				InstallationsRaw,
				MigratedInstallationsRaw,
				MaxInstallationsPerLogicalDatabase,
				CreateAt,
				DeleteAt,
				LockAcquiredBy,
				LockAcquiredAt
			FROM
			MultitenantDatabaseBackup2;
		`)
		if err != nil {
			return err
		}

		// The MultitenantDatabaseBackup2 table will be left for now just in case
		// and will be cleaned up in a future migration along with the previous
		// MultitenantDatabaseBackup table.

		return nil
	}},
	{semver.MustParse("0.32.0"), semver.MustParse("0.32.1"), func(e execer) error {
		_, err := e.Exec(`DROP TABLE MultitenantDatabaseBackup;`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`DROP TABLE MultitenantDatabaseBackup2;`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.32.1"), semver.MustParse("0.33.0"), func(e execer) error {
		// Add new tables for Provisioner Events:
		// - Event
		// - StateChangeEvent
		// - Subscription
		// - EventDelivery

		_, err := e.Exec(`
			CREATE TABLE Event (
				ID TEXT PRIMARY KEY,
				Timestamp BIGINT NOT NULL,
				EventType TEXT NOT NULL,
				ExtraData BYTEA NOT NULL
			);
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
			CREATE TABLE StateChangeEvent (
				ID TEXT PRIMARY KEY,
				EventID TEXT NOT NULL,
				ResourceID TEXT NOT NULL,
				ResourceType TEXT NOT NULL,
				OldState TEXT NOT NULL,
				NewState TEXT NOT NULL
			);
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`
			CREATE TABLE Subscription (
				ID TEXT PRIMARY KEY,
				Name TEXT NOT NULL,
				URL TEXT NOT NULL,
				OwnerID TEXT NOT NULL,
				EventType TEXT NOT NULL,
				LastDeliveryAttemptAt BIGINT NOT NULL,
				LastDeliveryStatus TEXT NOT NULL,
				FailureThreshold BIGINT NOT NULL,
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
			CREATE TABLE EventDelivery (
				ID TEXT PRIMARY KEY,
				EventID TEXT NOT NULL,
				SubscriptionID TEXT NOT NULL,
				Status TEXT NOT NULL,
				LastAttempt BIGINT NOT NULL,
				Attempts INT NOT NULL
			);
		`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.33.0"), semver.MustParse("0.34.0"), func(e execer) error {
		// Add PriorityEnv column to Installation
		_, err := e.Exec(`
				ALTER TABLE Installation
				ADD COLUMN PriorityEnvRaw BYTEA NULL;
		`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.34.0"), semver.MustParse("0.35.0"), func(e execer) error {
		// Create new table for Installation DNS records.
		// Drop DNS from Installation.

		_, err := e.Exec(`
			CREATE TABLE InstallationDNS (
				ID TEXT PRIMARY KEY,
				DomainName TEXT NOT NULL,
				InstallationID TEXT NOT NULL,
				IsPrimary BOOLEAN NOT NULL,
				CreateAt BIGINT NOT NULL,
				DeleteAt BIGINT NOT NULL
			);
		`)
		if err != nil {
			return err
		}

		_, err = e.Exec("ALTER TABLE Installation ADD COLUMN name TEXT;")
		if err != nil {
			return err
		}

		// For existing installations Name is set as first part of DNS.
		if e.DriverName() == driverPostgres {
			_, err = e.Exec("UPDATE Installation SET name = (SUBSTRING(dns, 0,  POSITION('.' in dns)));")
			if err != nil {
				return errors.Wrap(err, "failed to set name based on DNS")
			}
		} else if e.DriverName() == driverSqlite {
			_, err = e.Exec("UPDATE Installation SET name = (SUBSTR(DNS, 0, INSTR(DNS, '.')));")
			if err != nil {
				return err
			}
		}

		rows, err := e.Query("SELECT id, dns, createAt, deleteAt FROM INSTALLATION;")
		if err != nil {
			return errors.Wrap(err, "failed to fetch installation")
		}
		defer rows.Close()

		type installationDNS struct {
			installationID string
			dns            string
			createAt       int64
			deleteAt       int64
		}

		var existingRecs []installationDNS

		for rows.Next() {
			var installationID, dns string
			var createAt int64
			var deleteAt int64

			err = rows.Scan(&installationID, &dns, &createAt, &deleteAt)
			if err != nil {
				return errors.Wrap(err, "failed to scan rows")
			}

			existingRecs = append(existingRecs, installationDNS{
				installationID: installationID,
				dns:            dns,
				createAt:       createAt,
				deleteAt:       deleteAt,
			})
		}
		err = rows.Err()
		if err != nil {
			return errors.Wrap(err, "rows scanning error")
		}

		for _, dns := range existingRecs {
			_, err = e.Exec("INSERT INTO InstallationDNS(ID, DomainName, InstallationID, IsPrimary, CreateAt, DeleteAt) VALUES ($1, $2, $3, $4, $5, $6);",
				model.NewID(),
				dns.dns,
				dns.installationID,
				true,
				dns.createAt,
				dns.deleteAt)
			if err != nil {
				return errors.Wrap(err, "failed to insert installation DNS")
			}
		}

		if e.DriverName() == driverPostgres {
			_, err = e.Exec("ALTER TABLE Installation DROP COLUMN DNS;")
			if err != nil {
				return err
			}

			_, err = e.Exec("ALTER TABLE Installation ALTER COLUMN name SET NOT NULL;")
			if err != nil {
				return errors.Wrap(err, "failed to remove not null name constraint")
			}
		} else if e.DriverName() == driverSqlite {
			// We DROP DNS here and add NOT NULL for Name
			_, err = e.Exec(`ALTER TABLE Installation RENAME TO InstallationTemp;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`
					CREATE TABLE Installation (
						ID TEXT PRIMARY KEY,
						OwnerID TEXT NOT NULL,
						Name TEXT NOT NULL,
						Version TEXT NOT NULL,
						Image TEXT NOT NULL,
						Database TEXT NOT NULL,
						Filestore TEXT NOT NULL,
						License TEXT NULL,
						Size TEXT NOT NULL,
						MattermostEnvRaw BYTEA NULL,
						PriorityEnvRaw BYTEA NULL,
						Affinity TEXT NOT NULL,
						GroupSequence BIGINT NULL,
						GroupID TEXT NULL,
						State TEXT NOT NULL,
						APISecurityLock NOT NULL DEFAULT FALSE,
						SingleTenantDatabaseConfigRaw BYTEA NULL,
						ExternalDatabaseConfigRaw BYTEA NULL,
						CRVersion TEXT NOT NULL DEFAULT 'mattermost.com/v1alpha1',
						CreateAt BIGINT NOT NULL,
						DeleteAt BIGINT NOT NULL,
						DeletionPendingExpiry BIGINT NOT NULL,
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
						Name,
						Version,
						Image,
						Database,
						Filestore,
						License,
						Size,
						MattermostEnvRaw,
						PriorityEnvRaw,
						Affinity,
						GroupSequence,
						GroupID,
						State,
						APISecurityLock,
						SingleTenantDatabaseConfigRaw,
						ExternalDatabaseConfigRaw,
						CRVersion,
						CreateAt,
						DeleteAt,
						'0',
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

		_, err = e.Exec("CREATE UNIQUE INDEX Installation_Name_DeleteAt ON Installation (Name, DeleteAt);")
		if err != nil {
			return errors.Wrap(err, "failed to create InstallationDNS constraint")
		}

		_, err = e.Exec("CREATE UNIQUE INDEX InstallationDNS_DomainName_DeleteAt ON InstallationDNS (DomainName, DeleteAt);")
		if err != nil {
			return errors.Wrap(err, "failed to create InstallationDNS constraint")
		}

		// Make sure only one DNS for Installation is primary.
		_, err = e.Exec("CREATE UNIQUE INDEX InstallationDNS_IsPrimary_Installation_ID ON InstallationDNS (InstallationID) WHERE IsPrimary=True;")
		if err != nil {
			return errors.Wrap(err, "failed to create InstallationDNS constraint")
		}

		return nil
	}},
	{semver.MustParse("0.35.0"), semver.MustParse("0.36.0"), func(e execer) error {
		// Add GroupAnnotation table.

		_, err := e.Exec(`
			CREATE TABLE GroupAnnotation (
				ID TEXT PRIMARY KEY,
				GroupID TEXT NOT NULL,
				AnnotationID TEXT NOT NULL
			);
		`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.36.0"), semver.MustParse("0.37.0"), func(e execer) error {
		// Add unique constraint for GroupID and AnnotationID for GroupAnnotation.

		_, err := e.Exec(`
			CREATE UNIQUE INDEX GroupAnnotation_GroupID_AnnotationID ON GroupAnnotation (GroupID, AnnotationID);
		`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.37.0"), semver.MustParse("0.38.0"), func(e execer) error {
		// Add ExternalDatabaseConfigRaw column for installations.

		_, err := e.Exec(`
				ALTER TABLE Installation
				ADD COLUMN ExternalDatabaseConfigRaw BYTEA NULL;
				`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.38.0"), semver.MustParse("0.39.0"), func(e execer) error {
		// Add index on InstallationDNS(InstallationID) to avoid full table scan
		// for fetching all DNS records for single Installation.

		_, err := e.Exec(`
			CREATE INDEX ix_InstallationDNS_InstallationID on InstallationDNS(InstallationID)
		`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.39.0"), semver.MustParse("0.40.0"), func(e execer) error {
		// Add index on Installation(DeleteAt)
		_, err := e.Exec(`CREATE INDEX ix_Installation_DeleteAt on Installation(DeleteAt)`)
		if err != nil {
			return err
		}

		// Add index on Installation(State)
		_, err = e.Exec(`CREATE INDEX ix_Installation_State on Installation(State)`)
		if err != nil {
			return err
		}

		// Add index on StateChangeEvent(EventID)
		_, err = e.Exec(`CREATE INDEX ix_StateChangeEvent_EventID on StateChangeEvent(EventID)`)
		if err != nil {
			return err
		}

		// Add index on StateChangeEvent(ResourceID, NewState)
		_, err = e.Exec(`CREATE INDEX ix_StateChangeEvent_ResourceID_NewState on StateChangeEvent(ResourceID, NewState)`)
		if err != nil {
			return err
		}

		// Add index on Event(Timestamp)
		_, err = e.Exec(`CREATE INDEX ix_Event_Timestamp ON event (Timestamp)`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.40.0"), semver.MustParse("0.41.0"), func(e execer) error {
		// Add DeletionPendingExpiry column for installations.

		if e.DriverName() == driverPostgres {
			_, err := e.Exec(`
				ALTER TABLE Installation
				ADD COLUMN DeletionPendingExpiry BIGINT NOT NULL DEFAULT '0';
			`)
			if err != nil {
				return errors.Wrap(err, "failed to create DeletionPendingExpiry column")
			}

			_, err = e.Exec("ALTER TABLE Installation ALTER COLUMN DeletionPendingExpiry SET NOT NULL;")
			if err != nil {
				return errors.Wrap(err, "failed to remove not null expiry constraint")
			}
		} else if e.DriverName() == driverSqlite {
			// We DROP DNS here and add NOT NULL for Name
			_, err := e.Exec(`ALTER TABLE Installation RENAME TO InstallationTemp;`)
			if err != nil {
				return err
			}

			_, err = e.Exec(`
				CREATE TABLE Installation (
					ID TEXT PRIMARY KEY,
					OwnerID TEXT NOT NULL,
					Name TEXT NOT NULL,
					Version TEXT NOT NULL,
					Image TEXT NOT NULL,
					Database TEXT NOT NULL,
					Filestore TEXT NOT NULL,
					License TEXT NULL,
					Size TEXT NOT NULL,
					MattermostEnvRaw BYTEA NULL,
					PriorityEnvRaw BYTEA NULL,
					Affinity TEXT NOT NULL,
					GroupSequence BIGINT NULL,
					GroupID TEXT NULL,
					State TEXT NOT NULL,
					APISecurityLock NOT NULL DEFAULT FALSE,
					SingleTenantDatabaseConfigRaw BYTEA NULL,
					ExternalDatabaseConfigRaw BYTEA NULL,
					CRVersion TEXT NOT NULL DEFAULT 'mattermost.com/v1alpha1',
					CreateAt BIGINT NOT NULL,
					DeleteAt BIGINT NOT NULL,
					DeletionPendingExpiry BIGINT NOT NULL,
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
					Name,
					Version,
					Image,
					Database,
					Filestore,
					License,
					Size,
					MattermostEnvRaw,
					PriorityEnvRaw,
					Affinity,
					GroupSequence,
					GroupID,
					State,
					APISecurityLock,
					SingleTenantDatabaseConfigRaw,
					ExternalDatabaseConfigRaw,
					CRVersion,
					CreateAt,
					DeleteAt,
					'0',
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

			// Recreate indexes.
			_, err = e.Exec("CREATE UNIQUE INDEX Installation_Name_DeleteAt ON Installation (Name, DeleteAt);")
			if err != nil {
				return err
			}
			_, err = e.Exec(`CREATE INDEX ix_Installation_DeleteAt on Installation(DeleteAt)`)
			if err != nil {
				return err
			}
			_, err = e.Exec(`CREATE INDEX ix_Installation_State on Installation(State)`)
			if err != nil {
				return err
			}
		}

		return nil
	}},
	{semver.MustParse("0.41.0"), semver.MustParse("0.42.0"), func(e execer) error {
		fieldType := "JSONB"
		if e.DriverName() == driverSqlite {
			fieldType = "TEXT"
		}

		_, err := e.Exec(fmt.Sprintf(`ALTER TABLE Webhooks ADD COLUMN Headers %s NULL;`, fieldType))
		if err != nil {
			return err
		}

		_, err = e.Exec(fmt.Sprintf(`ALTER TABLE Subscription ADD COLUMN Headers %s NULL;`, fieldType))
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.42.0"), semver.MustParse("0.43.0"), func(e execer) error {
		// No-op migration to ensure SQLite is no longer used.
		if e.DriverName() == driverSqlite {
			panic("SQLite is no longer supported as a database backend")
		}

		return nil
	}},
	{semver.MustParse("0.43.0"), semver.MustParse("0.44.0"), func(e execer) error {
		_, err := e.Exec(`ALTER TABLE Installation ADD COLUMN DeletionLocked BOOLEAN NOT NULL DEFAULT 'false';`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.44.0"), semver.MustParse("0.45.0"), func(e execer) error {
		// Add AllowedIPRanges column for installations.

		_, err := e.Exec(`
				ALTER TABLE Installation ADD COLUMN AllowedIPRanges TEXT DEFAULT '';
				`)
		if err != nil {
			return errors.Wrap(err, "failed to add AllowedIPRanges column")
		}

		return nil
	}},
	{semver.MustParse("0.45.0"), semver.MustParse("0.46.0"), func(e execer) error {

		_, err := e.Exec(`ALTER TABLE Installation DROP COLUMN AllowedIPRanges;`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`ALTER TABLE Installation ADD COLUMN AllowedIPRanges JSON DEFAULT NULL;`)

		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.46.0"), semver.MustParse("0.47.0"), func(e execer) error {
		_, err := e.Exec(`ALTER TABLE Cluster ADD COLUMN PgBouncerConfig JSON`)
		if err != nil {
			return err
		}

		// Set some default config for all clusters.
		defaultPgBouncerConfig := model.NewDefaultPgBouncerConfig()
		value, err := defaultPgBouncerConfig.Value()
		if err != nil {
			return err
		}
		_, err = e.Exec(`UPDATE Cluster SET PgBouncerConfig = $1;`, value)
		if err != nil {
			return err
		}

		// Prevent null configs in the future.
		_, err = e.Exec(`ALTER TABLE Cluster ALTER COLUMN PgBouncerConfig SET NOT NULL;`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.47.0"), semver.MustParse("0.48.0"), func(e execer) error {
		_, err := e.Exec(`ALTER TABLE Cluster ADD COLUMN SchedulingLockAcquiredBy TEXT NULL;`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`ALTER TABLE Cluster ADD COLUMN SchedulingLockAcquiredAt BIGINT;`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`UPDATE Cluster SET SchedulingLockAcquiredAt = 0;`)
		if err != nil {
			return err
		}

		_, err = e.Exec(`ALTER TABLE Cluster ALTER COLUMN SchedulingLockAcquiredAt SET NOT NULL;`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.48.0"), semver.MustParse("0.49.0"), func(e execer) error {
		_, err := e.Exec(`ALTER TABLE Cluster ADD COLUMN Name TEXT NOT NULL DEFAULT '';`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.49.0"), semver.MustParse("0.50.0"), func(e execer) error {
		_, err := e.Exec(`ALTER TABLE Installation ADD COLUMN Volumes JSON DEFAULT NULL;`)
		if err != nil {
			return err
		}

		return nil
	}},
	{semver.MustParse("0.50.0"), semver.MustParse("0.51.0"), func(e execer) error {
		_, err := e.Exec(`
			ALTER TABLE Installation
			ADD COLUMN ScheduledDeletionTime BIGINT NOT NULL DEFAULT '0';
		`)
		if err != nil {
			return errors.Wrap(err, "failed to create ScheduledDeletionTime column")
		}

		_, err = e.Exec("ALTER TABLE Installation ALTER COLUMN ScheduledDeletionTime SET NOT NULL;")
		if err != nil {
			return errors.Wrap(err, "failed to remove not null constraint")
		}

		return nil
	}},
	{semver.MustParse("0.51.0"), semver.MustParse("0.52.0"), func(e execer) error {
		_, err := e.Exec(`ALTER TABLE Installation ADD COLUMN PodProbeOverrides JSON DEFAULT NULL;`)
		if err != nil {
			return errors.Wrap(err, "failed to create PodProbeOverrides column")
		}

		return nil
	}},
}
