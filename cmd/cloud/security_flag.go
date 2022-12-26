package main

import "github.com/spf13/cobra"

func setSecurityFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
}

type securityFlags struct {
	serverAddress string
}

func (flags *securityFlags) addFlags(cmd *cobra.Command) {
	flags.serverAddress, _ = cmd.Flags().GetString("server")
}

func setSecurityClusterFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("cluster", "", "The id of the cluster.")
	_ = cmd.MarkPersistentFlagRequired("cluster")
}

type securityClusterFlags struct {
	securityFlags
	clusterID string
}

func (flags *securityClusterFlags) addFlags(cmd *cobra.Command) {
	flags.securityFlags.addFlags(cmd)
	flags.clusterID, _ = cmd.Flags().GetString("cluster")
}

func setSecurityInstallationFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("installation", "", "The id of the installation.")
	_ = cmd.MarkPersistentFlagRequired("installation")
}

type securityInstallationFlags struct {
	securityFlags
	installationID string
}

func (flags *securityInstallationFlags) addFlags(cmd *cobra.Command) {
	flags.installationID, _ = cmd.Flags().GetString("installation")
}

func setSecurityClusterInstallationFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("cluster-installation", "", "The id of the cluster installation.")
	_ = cmd.MarkPersistentFlagRequired("cluster-installation")
}

type securityClusterInstallationFlags struct {
	securityFlags
	clusterInstallationID string
}

func (flags *securityClusterInstallationFlags) addFlags(cmd *cobra.Command) {
	flags.clusterInstallationID, _ = cmd.Flags().GetString("cluster-installation")
}

func setSecurityGroupFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("group", "", "The id of the group.")
	_ = cmd.MarkPersistentFlagRequired("group")
}

type securityGroupFlags struct {
	securityFlags
	groupID string
}

func (flags *securityGroupFlags) addFlags(cmd *cobra.Command) {
	flags.groupID, _ = cmd.Flags().GetString("group")
}

func setSecurityBackupFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("backup", "", "The id of the backup.")
	_ = cmd.MarkPersistentFlagRequired("backup")
}

type securityBackupFlags struct {
	securityFlags
	backupID string
}

func (flags *securityBackupFlags) addFlags(cmd *cobra.Command) {
	flags.backupID, _ = cmd.Flags().GetString("backup")
}
