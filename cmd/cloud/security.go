// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newCmdSecurity() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "security",
		Short: "Manage security locks for different cloud resources.",
	}

	setSecurityFlags(cmd)

	cmd.AddCommand(newCmdSecurityCluster())
	cmd.AddCommand(newCmdSecurityInstallation())
	cmd.AddCommand(newCmdSecurityClusterInstallation())
	cmd.AddCommand(newCmdSecurityGroup())
	cmd.AddCommand(newCmdSecurityBackup())

	return cmd
}

func newCmdSecurityCluster() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Manage security locks for cluster resources.",
	}

	setSecurityClusterFlags(cmd)

	cmd.AddCommand(newCmdSecurityClusterLock())
	cmd.AddCommand(newCmdSecurityClusterUnlock())

	return cmd
}

func newCmdSecurityClusterLock() *cobra.Command {

	var flags securityClusterFlags

	cmd := &cobra.Command{
		Use:   "api-lock",
		Short: "Lock API changes on a given cluster",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)
			if err := client.LockAPIForCluster(flags.clusterID); err != nil {
				return errors.Wrap(err, "failed to lock cluster API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.securityFlags.addFlags(cmd)
			flags.addFlags(cmd)
		},
	}

	return cmd
}

func newCmdSecurityClusterUnlock() *cobra.Command {

	var flags securityClusterFlags

	cmd := &cobra.Command{
		Use:   "api-unlock",
		Short: "Unlock API changes on a given cluster",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)
			if err := client.UnlockAPIForCluster(flags.clusterID); err != nil {
				return errors.Wrap(err, "failed to unlock cluster API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.securityFlags.addFlags(cmd)
			flags.addFlags(cmd)
		},
	}

	return cmd
}

func newCmdSecurityInstallation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "installation",
		Short: "Manage security locks for installation resources.",
	}

	setSecurityInstallationFlags(cmd)

	cmd.AddCommand(newCmdSecurityInstallationLock())
	cmd.AddCommand(newCmdSecurityInstallationUnlock())
	cmd.AddCommand(newCmdSecurityInstallationDeletionLock())
	cmd.AddCommand(newCmdSecurityInstallationDeletionUnlock())

	return cmd
}

func newCmdSecurityInstallationLock() *cobra.Command {

	var flags securityInstallationFlags

	cmd := &cobra.Command{
		Use:   "api-lock",
		Short: "Lock API changes on a given installation",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)
			if err := client.LockAPIForInstallation(flags.installationID); err != nil {
				return errors.Wrap(err, "failed to lock installation API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.securityFlags.addFlags(cmd)
			flags.addFlags(cmd)
		},
	}

	return cmd
}

func newCmdSecurityInstallationUnlock() *cobra.Command {

	var flags securityInstallationFlags

	cmd := &cobra.Command{
		Use:   "api-unlock",
		Short: "Unlock API changes on a given installation",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)
			if err := client.UnlockAPIForInstallation(flags.installationID); err != nil {
				return errors.Wrap(err, "failed to unlock installation API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.securityFlags.addFlags(cmd)
			flags.addFlags(cmd)
		},
	}

	return cmd
}

func newCmdSecurityInstallationDeletionUnlock() *cobra.Command {

	var flags securityInstallationFlags

	cmd := &cobra.Command{
		Use:   "deletion-unlock",
		Short: "Unlock deletion lock on installation, allowing it to be deleted",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)
			if err := client.UnlockDeletionLockForInstallation(flags.installationID); err != nil {
				return errors.Wrap(err, "failed to unlock installation deletion lock")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.securityFlags.addFlags(cmd)
			flags.addFlags(cmd)
		},
	}

	return cmd
}

func newCmdSecurityInstallationDeletionLock() *cobra.Command {
	var flags securityInstallationFlags

	cmd := &cobra.Command{
		Use:   "deletion-lock",
		Short: "Lock deletion lock on installation, preventing it from being deleted until unlocked",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)
			if err := client.LockDeletionLockForInstallation(flags.installationID); err != nil {
				return errors.Wrap(err, "failed to lock installation deletion lock")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.securityFlags.addFlags(cmd)
			flags.addFlags(cmd)
		},
	}

	return cmd
}

func newCmdSecurityClusterInstallation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster-installation",
		Short: "Manage security locks for cluster installation resources.",
	}

	setSecurityClusterInstallationFlags(cmd)

	cmd.AddCommand(newCmdSecurityClusterInstallationLock())
	cmd.AddCommand(newCmdSecurityClusterInstallationUnlock())

	return cmd
}

func newCmdSecurityClusterInstallationLock() *cobra.Command {

	var flags securityClusterInstallationFlags

	cmd := &cobra.Command{
		Use:   "api-lock",
		Short: "Lock API changes on a given cluster installation",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)
			if err := client.LockAPIForClusterInstallation(flags.clusterInstallationID); err != nil {
				return errors.Wrap(err, "failed to lock cluster installation API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.securityFlags.addFlags(cmd)
			flags.addFlags(cmd)
		},
	}

	return cmd
}

func newCmdSecurityClusterInstallationUnlock() *cobra.Command {

	var flags securityClusterInstallationFlags

	cmd := &cobra.Command{
		Use:   "api-unlock",
		Short: "Unlock API changes on a given cluster installation",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)
			if err := client.UnlockAPIForClusterInstallation(flags.clusterInstallationID); err != nil {
				return errors.Wrap(err, "failed to unlock cluster installation API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.securityFlags.addFlags(cmd)
			flags.addFlags(cmd)
		},
	}

	return cmd
}

func newCmdSecurityGroup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "group",
		Short: "Manage security locks for group resources.",
	}

	setSecurityGroupFlags(cmd)

	cmd.AddCommand(newCmdSecurityGroupLock())
	cmd.AddCommand(newCmdSecurityGroupUnlock())

	return cmd
}

func newCmdSecurityGroupLock() *cobra.Command {

	var flags securityGroupFlags

	cmd := &cobra.Command{
		Use:   "api-lock",
		Short: "Lock API changes on a given group",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)
			if err := client.LockAPIForGroup(flags.groupID); err != nil {
				return errors.Wrap(err, "failed to lock group API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.securityFlags.addFlags(cmd)
			flags.addFlags(cmd)
		},
	}

	return cmd
}

func newCmdSecurityGroupUnlock() *cobra.Command {

	var flags securityGroupFlags

	cmd := &cobra.Command{
		Use:   "api-unlock",
		Short: "Unlock API changes on a given group",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)
			if err := client.UnlockAPIForGroup(flags.groupID); err != nil {
				return errors.Wrap(err, "failed to unlock group API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.securityFlags.addFlags(cmd)
			flags.addFlags(cmd)
		},
	}

	return cmd
}

func newCmdSecurityBackup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage security locks for backup resources.",
	}

	setSecurityBackupFlags(cmd)

	cmd.AddCommand(newCmdSecurityBackupLock())
	cmd.AddCommand(newCmdSecurityBackupUnlock())

	return cmd
}

func newCmdSecurityBackupLock() *cobra.Command {

	var flags securityBackupFlags

	cmd := &cobra.Command{
		Use:   "api-lock",
		Short: "Lock API changes on a given backup",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)
			if err := client.LockAPIForBackup(flags.backupID); err != nil {
				return errors.Wrap(err, "failed to lock backup API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.securityFlags.addFlags(cmd)
			flags.addFlags(cmd)
		},
	}

	return cmd
}

func newCmdSecurityBackupUnlock() *cobra.Command {

	var flags securityBackupFlags

	cmd := &cobra.Command{
		Use:   "api-unlock",
		Short: "Unlock API changes on a given backup",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)
			if err := client.UnlockAPIForBackup(flags.backupID); err != nil {
				return errors.Wrap(err, "failed to unlock backup API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.securityFlags.addFlags(cmd)
			flags.addFlags(cmd)
		},
	}

	return cmd
}
