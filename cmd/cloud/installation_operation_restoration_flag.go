package main

import "github.com/spf13/cobra"

type installationRestorationRequestFlags struct {
	clusterFlags
	installationID string
	backupID       string
}

func (flags *installationRestorationRequestFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to be restored.")
	cmd.Flags().StringVar(&flags.backupID, "backup", "", "The id of the backup to restore.")
	_ = cmd.MarkFlagRequired("installation")
	_ = cmd.MarkFlagRequired("backup")
}

type installationRestorationsListFlags struct {
	clusterFlags
	pagingFlags
	tableOptions
	installationID        string
	clusterInstallationID string
	state                 string
}

func (flags *installationRestorationsListFlags) addFlags(cmd *cobra.Command) {
	flags.pagingFlags.addFlags(cmd)
	flags.tableOptions.addFlags(cmd)

	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to query operations.")
	cmd.Flags().StringVar(&flags.clusterInstallationID, "cluster-installation", "", "The cluster installation to filter operations by.")
	cmd.Flags().StringVar(&flags.state, "state", "", "The state to filter operations by.")
}

type installationRestorationGetFlags struct {
	clusterFlags
	restorationID string
}

func (flags *installationRestorationGetFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.restorationID, "restoration", "", "The id of restoration operation.")
	_ = cmd.MarkFlagRequired("restoration")
}
