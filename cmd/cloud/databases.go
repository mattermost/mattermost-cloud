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
	databaseCmd.AddCommand(multitenantDatabaseCmd)
	databaseCmd.AddCommand(logicalDatabaseCmd)
	databaseCmd.AddCommand(databaseSchemaCmd)

	// Multitenant Databases
	multitenantDatabaseListCmd.Flags().String("vpc-id", "", "The VPC ID by which to filter mulitenant databases.")
	multitenantDatabaseListCmd.Flags().String("database-type", "", "The database type by which to filter mulitenant databases.")
	registerPagingFlags(multitenantDatabaseListCmd)

	multitenantDatabaseGetCmd.Flags().String("multitenant-database", "", "The id of the mulitenant database to be fetched.")
	multitenantDatabaseGetCmd.MarkFlagRequired("multitenant-database")

	multitenantDatabaseUpdateCmd.Flags().String("multitenant-database", "", "The id of the mulitenant database to be updated.")
	multitenantDatabaseUpdateCmd.Flags().Int64("max-installations-per-logical-db", 10, "The maximum number of installations permitted in a single logical database (only applies to proxy databases).")
	multitenantDatabaseUpdateCmd.MarkFlagRequired("multitenant-database")

	multitenantDatabaseCmd.AddCommand(multitenantDatabaseListCmd)
	multitenantDatabaseCmd.AddCommand(multitenantDatabaseGetCmd)
	multitenantDatabaseCmd.AddCommand(multitenantDatabaseUpdateCmd)

	// Logical Databases
	logicalDatabaseListCmd.Flags().String("multitenant-database-id", "", "The multitenant database ID by which to filter logical databases.")
	registerPagingFlags(logicalDatabaseListCmd)

	logicalDatabaseGetCmd.Flags().String("logical-database", "", "The id of the logical database to be fetched.")
	logicalDatabaseGetCmd.MarkFlagRequired("logical-database")

	logicalDatabaseCmd.AddCommand(logicalDatabaseListCmd)
	logicalDatabaseCmd.AddCommand(logicalDatabaseGetCmd)

	// Database Schemas
	databaseSchemaListCmd.Flags().String("logical-database-id", "", "The logical database ID by which to filter database schemas.")
	databaseSchemaListCmd.Flags().String("installation-id", "", "The installation ID by which to filter database schemas.")
	registerPagingFlags(databaseSchemaListCmd)

	databaseSchemaGetCmd.Flags().String("database-schema", "", "The id of the database schema to be fetched.")
	databaseSchemaGetCmd.MarkFlagRequired("database-schema")

	databaseSchemaCmd.AddCommand(databaseSchemaListCmd)
	databaseSchemaCmd.AddCommand(databaseSchemaGetCmd)
}

var databaseCmd = &cobra.Command{
	Use:   "database",
	Short: "Manipulate database resources managed by the provisioning server.",
}

var multitenantDatabaseCmd = &cobra.Command{
	Use:   "multitenant",
	Short: "Manage and view multitenant database resources.",
}

var multitenantDatabaseListCmd = &cobra.Command{
	Use:   "list",
	Short: "List known multitenant databases.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		vpcID, _ := command.Flags().GetString("vpc-id")
		databaseType, _ := command.Flags().GetString("database-type")
		paging := parsePagingFlags(command)

		databases, err := client.GetMultitenantDatabases(&model.GetMultitenantDatabasesRequest{
			VpcID:        vpcID,
			DatabaseType: databaseType,
			Paging:       paging,
		})
		if err != nil {
			return errors.Wrap(err, "failed to query multitenant databases")
		}

		err = printJSON(databases)
		if err != nil {
			return err
		}

		return nil
	},
}

var multitenantDatabaseGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a particular multitenant database.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		multitenantDatabaseID, _ := command.Flags().GetString("multitenant-database")
		multitenantDatabase, err := client.GetMultitenantDatabase(multitenantDatabaseID)
		if err != nil {
			return errors.Wrap(err, "failed to query multitenant database")
		}
		if multitenantDatabase == nil {
			return nil
		}

		err = printJSON(multitenantDatabase)
		if err != nil {
			return err
		}

		return nil
	},
}

var multitenantDatabaseUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an multitenant database's configuration",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		multitenantDatabaseID, _ := command.Flags().GetString("multitenant-database")
		request := &model.PatchMultitenantDatabaseRequest{
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

		multitenantDatabase, err := client.UpdateMultitenantDatabase(multitenantDatabaseID, request)
		if err != nil {
			return errors.Wrap(err, "failed to update multitenant database")
		}

		return printJSON(multitenantDatabase)
	},
}

var logicalDatabaseCmd = &cobra.Command{
	Use:   "logical",
	Short: "Manage and view logical database resources.",
}

var logicalDatabaseListCmd = &cobra.Command{
	Use:   "list",
	Short: "List logical databases.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		multitenantDatabaseID, _ := command.Flags().GetString("multitenant-database-id")
		paging := parsePagingFlags(command)

		logicalDatabases, err := client.GetLogicalDatabases(&model.GetLogicalDatabasesRequest{
			MultitenantDatabaseID: multitenantDatabaseID,
			Paging:                paging,
		})
		if err != nil {
			return errors.Wrap(err, "failed to query logical databases")
		}

		err = printJSON(logicalDatabases)
		if err != nil {
			return err
		}

		return nil
	},
}

var logicalDatabaseGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a particular logical database.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		logicalDatabaseID, _ := command.Flags().GetString("logical-database")
		logicalDatabase, err := client.GetLogicalDatabase(logicalDatabaseID)
		if err != nil {
			return errors.Wrap(err, "failed to query logical database")
		}
		if logicalDatabase == nil {
			return nil
		}

		err = printJSON(logicalDatabase)
		if err != nil {
			return err
		}

		return nil
	},
}

var databaseSchemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Manage and view database schema resources.",
}

var databaseSchemaListCmd = &cobra.Command{
	Use:   "list",
	Short: "List database schemas.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		logicalDatabaseID, _ := command.Flags().GetString("logical-database-id")
		installationID, _ := command.Flags().GetString("installation-id")
		paging := parsePagingFlags(command)

		logicalDatabases, err := client.GetDatabaseSchemas(&model.GetDatabaseSchemaRequest{
			LogicalDatabaseID: logicalDatabaseID,
			InstallationID:    installationID,
			Paging:            paging,
		})
		if err != nil {
			return errors.Wrap(err, "failed to query database schemas")
		}

		err = printJSON(logicalDatabases)
		if err != nil {
			return err
		}

		return nil
	},
}

var databaseSchemaGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a particular database schema.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		databaseSchemaID, _ := command.Flags().GetString("database-schema")
		databaseSchema, err := client.GetDatabaseSchema(databaseSchemaID)
		if err != nil {
			return errors.Wrap(err, "failed to query database schema")
		}
		if databaseSchema == nil {
			return nil
		}

		err = printJSON(databaseSchema)
		if err != nil {
			return err
		}

		return nil
	},
}
