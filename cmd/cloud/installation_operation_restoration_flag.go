package main

import "github.com/spf13/cobra"

type installationRestorationRequestFlags struct {
	clusterFlags
	installationID string
	backupID       string
}

func (flags *installationRestorationRequestFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to be restored.")
	command.Flags().StringVar(&flags.backupID, "backup", "", "The id of the backup to restore.")
	_ = command.MarkFlagRequired("installation")
	_ = command.MarkFlagRequired("backup")
}

type installationRestorationsListFlags struct {
	clusterFlags
	pagingFlags
	tableOptions
	installationID        string
	clusterInstallationID string
	state                 string
}

func (flags *installationRestorationsListFlags) addFlags(command *cobra.Command) {
	flags.pagingFlags.addFlags(command)
	flags.tableOptions.addFlags(command)

	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to query operations.")
	command.Flags().StringVar(&flags.clusterInstallationID, "cluster-installation", "", "The cluster installation to filter operations by.")
	command.Flags().StringVar(&flags.state, "state", "", "The state to filter operations by.")
}

type installationRestorationGetFlags struct {
	clusterFlags
	restorationID string
}

func (flags *installationRestorationGetFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.restorationID, "restoration", "", "The id of restoration operation.")
	_ = command.MarkFlagRequired("restoration")
}
