// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newCmdInstallationRestorationOperation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "restoration",
		Short: "Manipulate installation restoration operations managed by the provisioning server.",
	}

	cmd.AddCommand(newCmdInstallationRestorationRequest())
	cmd.AddCommand(newCmdInstallationRestorationsListCmd())
	cmd.AddCommand(newCmdInstallationRestorationGetCmd())

	return cmd
}

func newCmdInstallationRestorationRequest() *cobra.Command {
	var flags installationRestorationRequestFlags

	cmd := &cobra.Command{
		Use:   "request",
		Short: "Request database restoration",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := createClient(flags.clusterFlags)

			installationDTO, err := client.RestoreInstallationDatabase(flags.installationID, flags.backupID)
			if err != nil {
				return errors.Wrap(err, "failed to request installation database restoration")
			}

			return printJSON(installationDTO)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdInstallationRestorationsListCmd() *cobra.Command {

	var flags installationRestorationsListFlags

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installation database restoration operations",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeInstallationRestorationsList(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func executeInstallationRestorationsList(flags installationRestorationsListFlags) error {
	client := createClient(flags.clusterFlags)

	paging := getPaging(flags.pagingFlags)

	request := &model.GetInstallationDBRestorationOperationsRequest{
		Paging:                paging,
		InstallationID:        flags.installationID,
		ClusterInstallationID: flags.clusterInstallationID,
		State:                 flags.state,
	}

	dbRestorationOperations, err := client.GetInstallationDBRestorationOperations(request)
	if err != nil {
		return errors.Wrap(err, "failed to list installation database restoration operations")
	}

	if enabled, customCols := getTableOutputOption(flags.tableOptions); enabled {
		var keys []string
		var vals [][]string

		if len(customCols) > 0 {
			data := make([]interface{}, 0, len(dbRestorationOperations))
			for _, elem := range dbRestorationOperations {
				data = append(data, elem)
			}
			keys, vals, err = prepareTableData(customCols, data)
			if err != nil {
				return errors.Wrap(err, "failed to prepare table output")
			}
		} else {
			keys, vals = defaultDBRestorationOperationTableData(dbRestorationOperations)
		}

		printTable(keys, vals)
		return nil
	}

	return printJSON(dbRestorationOperations)
}

func defaultDBRestorationOperationTableData(ops []*model.InstallationDBRestorationOperation) ([]string, [][]string) {
	keys := []string{"ID", "INSTALLATION ID", "BACKUP ID", "STATE", "CLUSTER INSTALLATION ID", "TARGET INSTALLATION STATE", "REQUEST AT"}
	vals := make([][]string, 0, len(ops))

	for _, restoration := range ops {
		vals = append(vals, []string{
			restoration.ID,
			restoration.InstallationID,
			restoration.BackupID,
			string(restoration.State),
			restoration.ClusterInstallationID,
			restoration.TargetInstallationState,
			model.TimeFromMillis(restoration.RequestAt).Format("2006-01-02 15:04:05 -0700 MST"),
		})
	}
	return keys, vals
}

func newCmdInstallationRestorationGetCmd() *cobra.Command {
	var flags installationRestorationGetFlags

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Fetches given installation database restoration operation.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := createClient(flags.clusterFlags)
			restorationOperation, err := client.GetInstallationDBRestoration(flags.restorationID)
			if err != nil {
				return errors.Wrap(err, "failed to get installation database restoration")
			}

			return printJSON(restorationOperation)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}
