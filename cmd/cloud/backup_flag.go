// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import "github.com/spf13/cobra"

type installationBackupCreateFlags struct {
	clusterFlags
	installationID string
}

func (flags *installationBackupCreateFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The installation id to be backed up.")
	_ = command.MarkFlagRequired("installation")
}

type installationBackupListFlags struct {
	clusterFlags
	pagingFlags
	tableOptions
	installationID string
	state          string
}

func (flags *installationBackupListFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The installation id for which the backups should be listed.")
	command.Flags().StringVar(&flags.state, "state", "", "The state to filter backups by.")
	flags.pagingFlags.addFlags(command)
	flags.tableOptions.addFlags(command)
}

type installationBackupGetFlags struct {
	clusterFlags
	backupID string
}

func (flags *installationBackupGetFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.backupID, "backup", "", "The id of the backup to get.")
	_ = command.MarkFlagRequired("backup")
}

type installationBackupDeleteFlags struct {
	clusterFlags
	backupID string
}

func (flags *installationBackupDeleteFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.backupID, "backup", "", "The id of the backup to delete.")
	_ = command.MarkFlagRequired("backup")
}
