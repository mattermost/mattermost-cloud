// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newCmdInstallationDBMigrationOperation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "db-migration",
		Short: "Manipulate installation db migration operations managed by the provisioning server.",
	}

	cmd.AddCommand(newCmdInstallationDBMigrationRequest())
	cmd.AddCommand(newCmdInstallationDBMigrationsList())
	cmd.AddCommand(newCmdInstallationDBMigrationGet())
	cmd.AddCommand(newCmdInstallationDBMigrationCommit())
	cmd.AddCommand(newCmdInstallationDBMigrationRollback())

	return cmd
}

func newCmdInstallationDBMigrationRequest() *cobra.Command {

	var flags installationDBMigrationRequestFlags

	cmd := &cobra.Command{
		Use:   "request",
		Short: "Request database migration to different DB",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := model.NewClient(flags.serverAddress)

			request := &model.InstallationDBMigrationRequest{
				InstallationID:         flags.installationID,
				DestinationDatabase:    flags.destinationDB,
				DestinationMultiTenant: &model.MultiTenantDBMigrationData{DatabaseID: flags.multiTenantDBID},
			}

			if flags.dryRun {
				if err := printJSON(request); err != nil {
					return errors.Wrap(err, "failed to print API request")
				}

				return nil
			}

			migrationOperation, err := client.MigrateInstallationDatabase(request)
			if err != nil {
				return errors.Wrap(err, "failed to request installation database migration")
			}

			if err = printJSON(migrationOperation); err != nil {
				return err
			}

			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdInstallationDBMigrationsList() *cobra.Command {
	var flags installationDBMigrationsListFlags

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installation database migration operations",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeInstallationDBMigrationsList(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd

}

func executeInstallationDBMigrationsList(flags installationDBMigrationsListFlags) error {
	client := model.NewClient(flags.serverAddress)

	paging := getPaging(flags.pagingFlags)

	request := &model.GetInstallationDBMigrationOperationsRequest{
		Paging:         paging,
		InstallationID: flags.installationID,
		State:          flags.state,
	}

	dbMigrationOperations, err := client.GetInstallationDBMigrationOperations(request)
	if err != nil {
		return errors.Wrap(err, "failed to list installation database migration operations")
	}

	if enabled, customCols := getTableOutputOption(flags.tableOptions); enabled {
		var keys []string
		var vals [][]string

		if len(customCols) > 0 {
			data := make([]interface{}, 0, len(dbMigrationOperations))
			for _, elem := range dbMigrationOperations {
				data = append(data, elem)
			}
			keys, vals, err = prepareTableData(customCols, data)
			if err != nil {
				return errors.Wrap(err, "failed to prepare table output")
			}
		} else {
			keys, vals = defaultDBMigrationOperationTableData(dbMigrationOperations)
		}

		printTable(keys, vals)
		return nil
	}

	if err = printJSON(dbMigrationOperations); err != nil {
		return err
	}

	return nil
}

func defaultDBMigrationOperationTableData(ops []*model.InstallationDBMigrationOperation) ([]string, [][]string) {
	keys := []string{"ID", "INSTALLATION ID", "STATE", "REQUEST AT"}
	vals := make([][]string, 0, len(ops))

	for _, migration := range ops {
		vals = append(vals, []string{
			migration.ID,
			migration.InstallationID,
			string(migration.State),
			model.TimeFromMillis(migration.RequestAt).Format("2006-01-02 15:04:05 -0700 MST"),
		})
	}
	return keys, vals
}

func newCmdInstallationDBMigrationGet() *cobra.Command {
	var flags installationDBMigrationGetFlags

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Fetches given installation database migration operation.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := model.NewClient(flags.serverAddress)

			migrationOperation, err := client.GetInstallationDBMigrationOperation(flags.dbMigrationID)
			if err != nil {
				return errors.Wrap(err, "failed to get installation database migration")
			}
			if err = printJSON(migrationOperation); err != nil {
				return err
			}

			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdInstallationDBMigrationCommit() *cobra.Command {
	var flags installationDBMigrationCommitFlags

	cmd := &cobra.Command{
		Use:   "commit",
		Short: "Commits database migration",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := model.NewClient(flags.serverAddress)

			migrationOperation, err := client.CommitInstallationDBMigration(flags.dbMigrationID)
			if err != nil {
				return errors.Wrap(err, "failed to commit installation database migration")
			}

			if err = printJSON(migrationOperation); err != nil {
				return err
			}

			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd

}

func newCmdInstallationDBMigrationRollback() *cobra.Command {
	var flags installationDBMigrationRollbackFlags

	cmd := &cobra.Command{
		Use:   "rollback",
		Short: "Triggers rollback of database migration",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := model.NewClient(flags.serverAddress)

			migrationOperation, err := client.RollbackInstallationDBMigration(flags.dbMigrationID)
			if err != nil {
				return errors.Wrap(err, "failed to trigger rollback of installation database migration")
			}

			if err = printJSON(migrationOperation); err != nil {
				return err
			}

			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd
}
