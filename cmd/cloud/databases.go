// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	databaseCmd.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
	databaseCmd.PersistentFlags().Bool("dry-run", false, "When set to true, only print the API request without sending it.")

	databaseListCmd.Flags().String("vpc-id", "", "The VPC ID by which to filter databases.")
	databaseListCmd.Flags().String("database-type", "", "The database type by which to filter databases.")
	registerPagingFlags(databaseListCmd)

	databaseUpdateCmd.Flags().String("database", "", "The id of the database to be updated.")
	databaseUpdateCmd.Flags().Int64("max-installations-per-logical-db", 10, "The maximum number of installations permitted in a single logical database (only applies to proxy databases).")
	databaseUpdateCmd.MarkFlagRequired("database")

	databaseCmd.AddCommand(databaseListCmd)
	databaseCmd.AddCommand(databaseUpdateCmd)
}

var databaseCmd = &cobra.Command{
	Use:   "database",
	Short: "View information on known external multitenant databases",
}

var databaseListCmd = &cobra.Command{
	Use:   "list",
	Short: "List multitenant databases that are currently in use.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		vpcID, _ := command.Flags().GetString("vpc-id")
		databaseType, _ := command.Flags().GetString("database-type")
		paging := parsePagingFlags(command)

		databases, err := client.GetMultitenantDatabases(&model.GetDatabasesRequest{
			VpcID:        vpcID,
			DatabaseType: databaseType,
			Paging:       paging,
		})
		if err != nil {
			return errors.Wrap(err, "failed to query databases")
		}

		err = printJSON(databases)
		if err != nil {
			return err
		}

		return nil
	},
}

var databaseUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an database's configuration",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		databaseID, _ := command.Flags().GetString("database")

		request := &model.PatchDatabaseRequest{
			MaxInstallationsPerLogicalDatabase: getInt64FlagPointer(command, "max-installations-per-logical-db"),
		}

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			err := printJSON(request)
			if err != nil {
				return errors.Wrap(err, "failed to print API request")
			}

			return nil
		}

		database, err := client.UpdateMultitenantDatabase(databaseID, request)
		if err != nil {
			return errors.Wrap(err, "failed to update database")
		}

		return printJSON(database)
	},
}
