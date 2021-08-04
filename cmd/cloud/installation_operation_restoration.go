// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"os"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	installationRestorationRequestCmd.Flags().String("installation", "", "The id of the installation to be restored.")
	installationRestorationRequestCmd.Flags().String("backup", "", "The id of the backup to restore.")
	installationRestorationRequestCmd.MarkFlagRequired("installation")
	installationRestorationRequestCmd.MarkFlagRequired("backup")

	installationRestorationsListCmd.Flags().String("installation", "", "The id of the installation to query operations.")
	installationRestorationsListCmd.Flags().String("state", "", "The state to filter operations by.")
	installationRestorationsListCmd.Flags().String("cluster-installation", "", "The cluster installation to filter operations by.")
	registerPagingFlags(installationRestorationsListCmd)
	installationRestorationsListCmd.Flags().Bool("table", false, "Whether to display output in a table or not.")

	installationRestorationGetCmd.Flags().String("restoration", "", "The id of restoration operation.")
	installationRestorationGetCmd.MarkFlagRequired("restoration")

	installationRestorationOperationCmd.AddCommand(installationRestorationRequestCmd)
	installationRestorationOperationCmd.AddCommand(installationRestorationsListCmd)
	installationRestorationOperationCmd.AddCommand(installationRestorationGetCmd)
}

var installationRestorationOperationCmd = &cobra.Command{
	Use:   "restoration",
	Short: "Manipulate installation restoration operations managed by the provisioning server.",
}

var installationRestorationRequestCmd = &cobra.Command{
	Use:   "request",
	Short: "Request database restoration",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		backupID, _ := command.Flags().GetString("backup")

		installationDTO, err := client.RestoreInstallationDatabase(installationID, backupID)
		if err != nil {
			return errors.Wrap(err, "failed to request installation database restoration")
		}

		err = printJSON(installationDTO)
		if err != nil {
			return err
		}

		return nil
	},
}

var installationRestorationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installation database restoration operations",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		clusterInstallationID, _ := command.Flags().GetString("cluster-installation")
		state, _ := command.Flags().GetString("state")
		paging := parsePagingFlags(command)

		request := &model.GetInstallationDBRestorationOperationsRequest{
			Paging:                paging,
			InstallationID:        installationID,
			ClusterInstallationID: clusterInstallationID,
			State:                 state,
		}

		dbRestorationOperations, err := client.GetInstallationDBRestorationOperations(request)
		if err != nil {
			return errors.Wrap(err, "failed to list installation database restoration operations")
		}

		outputToTable, _ := command.Flags().GetBool("table")
		if outputToTable {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.SetHeader([]string{"ID", "INSTALLATION ID", "BACKUP ID", "STATE", "CLUSTER INSTALLATION ID", "TARGET INSTALLATION STATE", "REQUEST AT"})

			for _, restoration := range dbRestorationOperations {
				table.Append([]string{
					restoration.ID,
					restoration.InstallationID,
					restoration.BackupID,
					string(restoration.State),
					restoration.ClusterInstallationID,
					restoration.TargetInstallationState,
					model.TimeFromMillis(restoration.RequestAt).Format("2006-01-02 15:04:05 -0700 MST"),
				})
			}
			table.Render()

			return nil
		}

		err = printJSON(dbRestorationOperations)
		if err != nil {
			return err
		}

		return nil
	},
}

var installationRestorationGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Fetches given installation database restoration operation.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		restorationID, _ := command.Flags().GetString("restoration")

		restorationOperation, err := client.GetInstallationDBRestoration(restorationID)
		if err != nil {
			return errors.Wrap(err, "failed to get installation database restoration")
		}

		err = printJSON(restorationOperation)
		if err != nil {
			return err
		}

		return nil
	},
}
