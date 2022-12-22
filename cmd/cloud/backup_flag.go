// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import "github.com/spf13/cobra"

type installationBackupCreateFlags struct {
	clusterFlags
	installationID string
}

func (flags *installationBackupCreateFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The installation id to be backed up.")
	_ = cmd.MarkFlagRequired("installation")
}

type installationBackupListFlags struct {
	clusterFlags
	pagingFlags
	tableOptions
	installationID string
	state          string
}

func (flags *installationBackupListFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The installation id for which the backups should be listed.")
	cmd.Flags().StringVar(&flags.state, "state", "", "The state to filter backups by.")
	flags.pagingFlags.addFlags(cmd)
	flags.tableOptions.addFlags(cmd)
}

type installationBackupGetFlags struct {
	clusterFlags
	backupID string
}

func (flags *installationBackupGetFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.backupID, "backup", "", "The id of the backup to get.")
	_ = cmd.MarkFlagRequired("backup")
}

type installationBackupDeleteFlags struct {
	clusterFlags
	backupID string
}

func (flags *installationBackupDeleteFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.backupID, "backup", "", "The id of the backup to delete.")
	_ = cmd.MarkFlagRequired("backup")
}
