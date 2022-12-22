// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"fmt"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func databaseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "database",
		Short: "Manipulate database resources managed by the provisioning server.",
	}

	setClusterFlags(cmd)

	cmd.AddCommand(databaseMultitenantCmd())
	cmd.AddCommand(databaseLogicalCmd())
	cmd.AddCommand(databaseSchemaCmd())

	return cmd
}

func databaseMultitenantCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "multitenant",
		Short: "Manage and view multitenant database resources.",
	}

	cmd.AddCommand(databaseMultitenantListCmd())
	cmd.AddCommand(databaseMultitenantGetCmd())
	cmd.AddCommand(databaseMultitenantUpdateCmd())
	cmd.AddCommand(databaseMultitenantDeleteCmd())
	cmd.AddCommand(databaseMultitenantReportCmd())

	return cmd
}

func databaseMultitenantListCmd() *cobra.Command {

	var flags databaseMultiTenantListFlag

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List known multitenant databases.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return executeDatabaseMultitenantListCmd(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func executeDatabaseMultitenantListCmd(flags databaseMultiTenantListFlag) error {
	client := model.NewClient(flags.serverAddress)

	paging := getPaging(flags.pagingFlags)

	multitenantDatabases, err := client.GetMultitenantDatabases(&model.GetMultitenantDatabasesRequest{
		VpcID:        flags.vpcID,
		DatabaseType: flags.databaseType,
		Paging:       paging,
	})
	if err != nil {
		return errors.Wrap(err, "failed to query multitenant databases")
	}

	if enabled, customCols := getTableOutputOption(flags.tableOptions); enabled {
		var keys []string
		var vals [][]string

		if len(customCols) > 0 {
			data := make([]interface{}, 0, len(multitenantDatabases))
			for _, mtd := range multitenantDatabases {
				data = append(data, mtd)
			}
			keys, vals, err = prepareTableData(customCols, data)
			if err != nil {
				return errors.Wrap(err, "failed to prepare table output")
			}
		} else {
			keys, vals = defaultMultitenantDatabaseTableData(multitenantDatabases)
		}

		printTable(keys, vals)
		return nil
	}

	return printJSON(multitenantDatabases)
}

func defaultMultitenantDatabaseTableData(multitenantDatabases []*model.MultitenantDatabase) ([]string, [][]string) {
	keys := []string{"ID", "RDS CLUSTER ID", "TYPE", "STATE", "INSTALLATIONS"}
	vals := make([][]string, 0, len(multitenantDatabases))
	for _, multitenantDatabase := range multitenantDatabases {
		vals = append(vals, []string{multitenantDatabase.ID, multitenantDatabase.RdsClusterID, multitenantDatabase.DatabaseType, multitenantDatabase.State, fmt.Sprintf("%d", multitenantDatabase.Installations.Count())})
	}
	return keys, vals
}

func databaseMultitenantGetCmd() *cobra.Command {
	var flags databaseMultiTenantGetFlag

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a particular multitenant database.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)

			multitenantDatabase, err := client.GetMultitenantDatabase(flags.multitenantDatabaseID)
			if err != nil {
				return errors.Wrap(err, "failed to query multitenant database")
			}
			if multitenantDatabase == nil {
				return nil
			}

			return printJSON(multitenantDatabase)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func databaseMultitenantUpdateCmd() *cobra.Command {
	var flags databaseMultiTenantUpdateFlag

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update an multitenant database's configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)

			request := &model.PatchMultitenantDatabaseRequest{}
			if flags.isMaxInstallationsChanged {
				request.MaxInstallationsPerLogicalDatabase = &flags.maxInstallations
			}

			if flags.dryRun {
				return runDryRun(request)
			}

			multitenantDatabase, err := client.UpdateMultitenantDatabase(flags.multitenantDatabaseID, request)
			if err != nil {
				return errors.Wrap(err, "failed to update multitenant database")
			}

			return printJSON(multitenantDatabase)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			flags.databaseMultiTenantUpdateFlagChanged.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func databaseMultitenantDeleteCmd() *cobra.Command {
	var flags databaseMultiTenantDeleteFlag

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete an multitenant database's configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)

			if err := client.DeleteMultitenantDatabase(flags.multitenantDatabaseID, flags.force); err != nil {
				return errors.Wrap(err, "failed to update multitenant database")
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

func databaseLogicalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logical",
		Short: "Manage and view logical database resources.",
	}

	cmd.AddCommand(databaseLogicalListCmd())
	cmd.AddCommand(databaseLogicalGetCmd())

	return cmd
}

func databaseLogicalListCmd() *cobra.Command {
	var flags databaseLogicalListFlag

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List logical databases.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return executeDatabaseLogicalListCmd(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func executeDatabaseLogicalListCmd(flags databaseLogicalListFlag) error {
	client := model.NewClient(flags.serverAddress)

	paging := getPaging(flags.pagingFlags)

	logicalDatabases, err := client.GetLogicalDatabases(&model.GetLogicalDatabasesRequest{
		MultitenantDatabaseID: flags.multitenantDatabaseID,
		Paging:                paging,
	})
	if err != nil {
		return errors.Wrap(err, "failed to query logical databases")
	}

	if enabled, customCols := getTableOutputOption(flags.tableOptions); enabled {
		var keys []string
		var vals [][]string

		if len(customCols) > 0 {
			data := make([]interface{}, 0, len(logicalDatabases))
			for _, ldb := range logicalDatabases {
				data = append(data, ldb)
			}
			keys, vals, err = prepareTableData(customCols, data)
			if err != nil {
				return errors.Wrap(err, "failed to prepare table output")
			}
		} else {
			keys, vals = defaultLogicalDatabaseTableData(logicalDatabases)
		}

		printTable(keys, vals)
		return nil
	}

	return printJSON(logicalDatabases)
}

func defaultLogicalDatabaseTableData(logicalDatabases []*model.LogicalDatabase) ([]string, [][]string) {
	keys := []string{"ID", "MULTITENANT DATABASE", "NAME"}
	vals := make([][]string, 0, len(logicalDatabases))
	for _, logicalDatabase := range logicalDatabases {
		vals = append(vals, []string{logicalDatabase.ID, logicalDatabase.MultitenantDatabaseID, logicalDatabase.Name})
	}
	return keys, vals
}

func databaseLogicalGetCmd() *cobra.Command {
	var flags databaseLogicalGetFlag

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a particular logical database.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)

			logicalDatabase, err := client.GetLogicalDatabase(flags.logicalDatabaseID)
			if err != nil {
				return errors.Wrap(err, "failed to query logical database")
			}
			if logicalDatabase == nil {
				return nil
			}

			return printJSON(logicalDatabase)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func databaseSchemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Manage and view database schema resources.",
	}

	cmd.AddCommand(databaseSchemaListCmd())
	cmd.AddCommand(databaseSchemaGetCmd())

	return cmd
}

func databaseSchemaListCmd() *cobra.Command {
	var flags databaseSchemaListFlag

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List database schemas.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return executeDatabaseSchemaListCmd(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func executeDatabaseSchemaListCmd(flags databaseSchemaListFlag) error {
	client := model.NewClient(flags.serverAddress)

	paging := getPaging(flags.pagingFlags)

	databaseSchemas, err := client.GetDatabaseSchemas(&model.GetDatabaseSchemaRequest{
		LogicalDatabaseID: flags.logicalDatabaseID,
		InstallationID:    flags.installationID,
		Paging:            paging,
	})
	if err != nil {
		return errors.Wrap(err, "failed to query database schemas")
	}

	if enabled, customCols := getTableOutputOption(flags.tableOptions); enabled {
		var keys []string
		var vals [][]string

		if len(customCols) > 0 {
			data := make([]interface{}, 0, len(databaseSchemas))
			for _, dbs := range databaseSchemas {
				data = append(data, dbs)
			}
			keys, vals, err = prepareTableData(customCols, data)
			if err != nil {
				return errors.Wrap(err, "failed to prepare table output")
			}
		} else {
			keys, vals = defaultDatabaseSchemaTableData(databaseSchemas)
		}

		printTable(keys, vals)
		return nil
	}

	return printJSON(databaseSchemas)
}

func defaultDatabaseSchemaTableData(databaseSchemas []*model.DatabaseSchema) ([]string, [][]string) {
	keys := []string{"ID", "LOGICAL DATABASE", "INSTALLATION", "NAME"}
	vals := make([][]string, 0, len(databaseSchemas))
	for _, databaseSchema := range databaseSchemas {
		vals = append(vals, []string{databaseSchema.ID, databaseSchema.LogicalDatabaseID, databaseSchema.InstallationID, databaseSchema.Name})
	}
	return keys, vals
}

func databaseSchemaGetCmd() *cobra.Command {
	var flags databaseSchemaGetFlag

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a particular database schema.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)

			databaseSchema, err := client.GetDatabaseSchema(flags.databaseSchemaID)
			if err != nil {
				return errors.Wrap(err, "failed to query database schema")
			}
			if databaseSchema == nil {
				return nil
			}

			return printJSON(databaseSchema)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func databaseMultitenantReportCmd() *cobra.Command {
	var flags databaseMultiTenantReportFlag

	cmd := &cobra.Command{
		Use:   "report",
		Short: "Get a report of deployment details for a given multitenant database",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return executeMultiTenantDatabaseReportCmd(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func executeMultiTenantDatabaseReportCmd(flags databaseMultiTenantReportFlag) error {
	client := model.NewClient(flags.serverAddress)

	multitenantDatabase, err := client.GetMultitenantDatabase(flags.multitenantDatabaseID)
	if err != nil {
		return errors.Wrap(err, "failed to query multitenant database")
	}
	if multitenantDatabase == nil {
		return nil
	}

	output := fmt.Sprintf("Multitenant Database: %s\n", multitenantDatabase.ID)
	output += fmt.Sprintf(" ├ Created: %s\n", multitenantDatabase.CreationDateString())
	output += fmt.Sprintf(" ├ State: %s\n", multitenantDatabase.State)
	output += fmt.Sprintf(" ├ Type: %s\n", multitenantDatabase.DatabaseType)
	output += fmt.Sprintf(" ├ VPC: %s\n", multitenantDatabase.VpcID)
	output += fmt.Sprintf(" ├ Installations: %d\n", multitenantDatabase.Installations.Count())
	output += fmt.Sprintf(" ├ Writer Endpoint: %s\n", multitenantDatabase.WriterEndpoint)
	output += fmt.Sprintf(" ├ Reader Endpoint: %s\n", multitenantDatabase.ReaderEndpoint)

	if multitenantDatabase.DatabaseType == model.DatabaseEngineTypePostgresProxy {
		logicalDatabases, err := client.GetLogicalDatabases(&model.GetLogicalDatabasesRequest{
			MultitenantDatabaseID: multitenantDatabase.ID,
			Paging:                model.AllPagesNotDeleted(),
		})
		if err != nil {
			return errors.Wrap(err, "failed to query installation logical databases")
		}

		output += fmt.Sprintf(" └ Logical Databases: %d\n", len(logicalDatabases))
		output += fmt.Sprintf("   └ Average Installations Per Logical Database: %.2f\n", float64(multitenantDatabase.Installations.Count())/float64(len(logicalDatabases)))
	}

	fmt.Println(output)

	return nil
}
