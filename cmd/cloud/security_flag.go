package main

import "github.com/spf13/cobra"

func setSecurityFlags(command *cobra.Command) {
	command.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
}

type securityFlags struct {
	clusterFlags
}

func (flags *securityFlags) addFlags(command *cobra.Command) {
	flags.serverAddress, _ = command.Flags().GetString("server")
}

func setSecurityClusterFlags(command *cobra.Command) {
	command.PersistentFlags().String("cluster", "", "The id of the cluster.")
	_ = command.MarkPersistentFlagRequired("cluster")
}

type securityClusterFlags struct {
	securityFlags
	clusterID string
}

func (flags *securityClusterFlags) addFlags(command *cobra.Command) {
	flags.securityFlags.addFlags(command)
	flags.clusterID, _ = command.Flags().GetString("cluster")
}

func setSecurityInstallationFlags(command *cobra.Command) {
	command.PersistentFlags().String("installation", "", "The id of the installation.")
	_ = command.MarkPersistentFlagRequired("installation")
}

type securityInstallationFlags struct {
	securityFlags
	installationID string
}

func (flags *securityInstallationFlags) addFlags(command *cobra.Command) {
	flags.installationID, _ = command.Flags().GetString("installation")
}

func setSecurityClusterInstallationFlags(command *cobra.Command) {
	command.PersistentFlags().String("cluster-installation", "", "The id of the cluster installation.")
	_ = command.MarkPersistentFlagRequired("cluster-installation")
}

type securityClusterInstallationFlags struct {
	securityFlags
	clusterInstallationID string
}

func (flags *securityClusterInstallationFlags) addFlags(command *cobra.Command) {
	flags.clusterInstallationID, _ = command.Flags().GetString("cluster-installation")
}

func setSecurityGroupFlags(command *cobra.Command) {
	command.PersistentFlags().String("group", "", "The id of the group.")
	_ = command.MarkPersistentFlagRequired("group")
}

type securityGroupFlags struct {
	securityFlags
	groupID string
}

func (flags *securityGroupFlags) addFlags(command *cobra.Command) {
	flags.groupID, _ = command.Flags().GetString("group")
}

func setSecurityBackupFlags(command *cobra.Command) {
	command.PersistentFlags().String("backup", "", "The id of the backup.")
	_ = command.MarkPersistentFlagRequired("backup")
}

type securityBackupFlags struct {
	securityFlags
	backupID string
}

func (flags *securityBackupFlags) addFlags(command *cobra.Command) {
	flags.backupID, _ = command.Flags().GetString("backup")
}
