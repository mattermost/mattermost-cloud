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
	backupCmd.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")

	backupCreateCmd.Flags().String("installation", "", "The installation id to be backed up.")
	backupCreateCmd.MarkFlagRequired("installation")

	backupListCmd.Flags().String("installation", "", "The installation id for which the backups should be listed.")
	backupListCmd.Flags().String("state", "", "The state to filter backups by.")
	registerPagingFlags(backupListCmd)
	backupListCmd.Flags().Bool("table", false, "Whether to display the returned backup list in a table or not.")

	backupGetCmd.Flags().String("backup", "", "The id of the backup to get.")
	backupGetCmd.MarkFlagRequired("backup")

	backupDeleteCmd.Flags().String("backup", "", "The id of the backup to delete.")
	backupDeleteCmd.MarkFlagRequired("backup")

	backupCmd.AddCommand(backupCreateCmd)
	backupCmd.AddCommand(backupListCmd)
	backupCmd.AddCommand(backupGetCmd)
	backupCmd.AddCommand(backupDeleteCmd)
}

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manipulate installation backups managed by the provisioning server.",
}

var backupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Request an installation backup.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")

		backup, err := client.CreateInstallationBackup(installationID)
		if err != nil {
			return errors.Wrap(err, "failed to request installation backup")
		}

		return printJSON(backup)
	},
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installation backups.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		clusterInstallationID, _ := command.Flags().GetString("cluster-installation")
		state, _ := command.Flags().GetString("state")
		paging := parsePagingFlags(command)

		request := &model.GetInstallationBackupsRequest{
			InstallationID:        installationID,
			ClusterInstallationID: clusterInstallationID,
			State:                 state,
			Paging:                paging,
		}

		backups, err := client.GetInstallationBackups(request)
		if err != nil {
			return errors.Wrap(err, "failed to get backup")
		}

		outputToTable, _ := command.Flags().GetBool("table")
		if outputToTable {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.SetHeader([]string{"ID", "INSTALLATION ID", "STATE", "CLUSTER INSTALLATION ID", "REQUEST AT"})

			for _, backup := range backups {
				table.Append([]string{
					backup.ID,
					backup.InstallationID,
					string(backup.State),
					backup.ClusterInstallationID,
					model.TimeFromMillis(backup.RequestAt).Format("2006-01-02 15:04:05 -0700 MST"),
				})
			}
			table.Render()

			return nil
		}

		err = printJSON(backups)
		if err != nil {
			return err
		}

		return nil
	},
}

var backupGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get installation backup.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		backupID, _ := command.Flags().GetString("backup")

		backup, err := client.GetInstallationBackup(backupID)
		if err != nil {
			return errors.Wrap(err, "failed to get backup")
		}

		err = printJSON(backup)
		if err != nil {
			return err
		}

		return nil
	},
}

var backupDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete installation backup.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		backupID, _ := command.Flags().GetString("backup")

		err := client.DeleteInstallationBackup(backupID)
		if err != nil {
			return errors.Wrap(err, "failed to delete backup")
		}

		return nil
	},
}
