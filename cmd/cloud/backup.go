// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"context"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newCmdInstallationBackup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manipulate installation backups managed by the provisioning server.",
	}

	cmd.AddCommand(newCmdInstallationBackupCreate())
	cmd.AddCommand(newCmdInstallationBackupList())
	cmd.AddCommand(newCmdInstallationBackupGet())
	cmd.AddCommand(newCmdInstallationBackupDelete())

	return cmd
}

func newCmdInstallationBackupCreate() *cobra.Command {
	var flags installationBackupCreateFlags

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Request an installation backup.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := createClient(command.Context(), flags.clusterFlags)

			backup, err := client.CreateInstallationBackup(flags.installationID)
			if err != nil {
				return errors.Wrap(err, "failed to request installation backup")
			}

			return printJSON(backup)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdInstallationBackupList() *cobra.Command {
	var flags installationBackupListFlags

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List installation backups.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeInstallationBackupListCmd(command.Context(), flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func executeInstallationBackupListCmd(ctx context.Context, flags installationBackupListFlags) error {
	client := createClient(ctx, flags.clusterFlags)

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

func newCmdInstallationBackupGet() *cobra.Command {
	var flags installationBackupGetFlags

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get installation backup.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := createClient(command.Context(), flags.clusterFlags)

			backup, err := client.GetInstallationBackup(flags.backupID)
			if err != nil {
				return errors.Wrap(err, "failed to get backup")
			}

			return printJSON(backup)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdInstallationBackupDelete() *cobra.Command {
	var flags installationBackupDeleteFlags

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete installation backup.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := createClient(command.Context(), flags.clusterFlags)

			if err := client.DeleteInstallationBackup(flags.backupID); err != nil {
				return errors.Wrap(err, "failed to delete backup")
			}

			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}
