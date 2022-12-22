// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func installationBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manipulate installation backups managed by the provisioning server.",
	}

	cmd.AddCommand(installationBackupCreateCmd())
	cmd.AddCommand(installationBackupListCmd())
	cmd.AddCommand(installationBackupGetCmd())
	cmd.AddCommand(installationBackupDeleteCmd())

	return cmd
}

func installationBackupCreateCmd() *cobra.Command {
	var flags installationBackupCreateFlags

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Request an installation backup.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			client := model.NewClient(flags.serverAddress)

			backup, err := client.CreateInstallationBackup(flags.installationID)
			if err != nil {
				return errors.Wrap(err, "failed to request installation backup")
			}

			return printJSON(backup)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func installationBackupListCmd() *cobra.Command {
	var flags installationBackupListFlags

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installation backups.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return executeInstallationBackupListCmd(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func executeInstallationBackupListCmd(flags installationBackupListFlags) error {
	client := model.NewClient(flags.serverAddress)

	paging := getPaging(flags.pagingFlags)

	request := &model.GetInstallationBackupsRequest{
		InstallationID: flags.installationID,
		State:          flags.state,
		Paging:         paging,
	}

	backups, err := client.GetInstallationBackups(request)
	if err != nil {
		return errors.Wrap(err, "failed to get backup")
	}

	if enabled, customCols := getTableOutputOption(flags.tableOptions); enabled {
		var keys []string
		var vals [][]string

		if len(customCols) > 0 {
			data := make([]interface{}, 0, len(backups))
			for _, elem := range backups {
				data = append(data, elem)
			}
			keys, vals, err = prepareTableData(customCols, data)
			if err != nil {
				return errors.Wrap(err, "failed to prepare table output")
			}
		} else {
			keys, vals = defaultBackupTableData(backups)
		}

		printTable(keys, vals)
		return nil
	}

	return printJSON(backups)
}

func defaultBackupTableData(backups []*model.InstallationBackup) ([]string, [][]string) {
	keys := []string{"ID", "INSTALLATION ID", "STATE", "CLUSTER INSTALLATION ID", "REQUEST AT"}
	vals := make([][]string, 0, len(backups))

	for _, backup := range backups {
		vals = append(vals, []string{
			backup.ID,
			backup.InstallationID,
			string(backup.State),
			backup.ClusterInstallationID,
			model.TimeFromMillis(backup.RequestAt).Format("2006-01-02 15:04:05 -0700 MST"),
		})
	}

	return keys, vals
}

func installationBackupGetCmd() *cobra.Command {
	var flags installationBackupGetFlags

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get installation backup.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			client := model.NewClient(flags.serverAddress)

			backup, err := client.GetInstallationBackup(flags.backupID)
			if err != nil {
				return errors.Wrap(err, "failed to get backup")
			}

			return printJSON(backup)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func installationBackupDeleteCmd() *cobra.Command {
	var flags installationBackupDeleteFlags

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete installation backup.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			client := model.NewClient(flags.serverAddress)

			if err := client.DeleteInstallationBackup(flags.backupID); err != nil {
				return errors.Wrap(err, "failed to delete backup")
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
