// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func securityCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "security",
		Short: "Manage security locks for different cloud resources.",
	}

	setSecurityFlags(cmd)

	cmd.AddCommand(securityClusterCmd())
	cmd.AddCommand(securityInstallationCmd())
	cmd.AddCommand(securityClusterInstallationCmd())
	cmd.AddCommand(securityGroupCmd())
	cmd.AddCommand(securityBackupCmd())

	return cmd
}

func securityClusterCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Manage security locks for cluster resources.",
	}

	setSecurityClusterFlags(cmd)

	cmd.AddCommand(securityClusterLockCmd())
	cmd.AddCommand(securityClusterUnlockCmd())

	return cmd
}

func securityClusterLockCmd() *cobra.Command {

	var flags securityClusterFlags

	cmd := &cobra.Command{
		Use:   "api-lock",
		Short: "Lock API changes on a given cluster",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			if err := client.LockAPIForCluster(flags.clusterID); err != nil {
				return errors.Wrap(err, "failed to lock cluster API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.addFlags(cmd)
			return
		},
	}

	return cmd
}

func securityClusterUnlockCmd() *cobra.Command {

	var flags securityClusterFlags

	cmd := &cobra.Command{
		Use:   "api-unlock",
		Short: "Unlock API changes on a given cluster",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			if err := client.UnlockAPIForCluster(flags.clusterID); err != nil {
				return errors.Wrap(err, "failed to unlock cluster API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.addFlags(cmd)
			return
		},
	}

	return cmd
}

func securityInstallationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "installation",
		Short: "Manage security locks for installation resources.",
	}

	setSecurityInstallationFlags(cmd)

	cmd.AddCommand(securityInstallationLockCmd())
	cmd.AddCommand(securityInstallationUnlockCmd())

	return cmd
}

func securityInstallationLockCmd() *cobra.Command {

	var flags securityInstallationFlags

	cmd := &cobra.Command{
		Use:   "api-lock",
		Short: "Lock API changes on a given installation",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			if err := client.LockAPIForInstallation(flags.installationID); err != nil {
				return errors.Wrap(err, "failed to lock installation API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.addFlags(cmd)
			return
		},
	}

	return cmd
}

func securityInstallationUnlockCmd() *cobra.Command {

	var flags securityInstallationFlags

	cmd := &cobra.Command{
		Use:   "api-unlock",
		Short: "Unlock API changes on a given installation",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			if err := client.UnlockAPIForInstallation(flags.installationID); err != nil {
				return errors.Wrap(err, "failed to unlock installation API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.addFlags(cmd)
			return
		},
	}

	return cmd
}

func securityClusterInstallationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "cluster-installation",
		Short: "Manage security locks for cluster installation resources.",
	}

	setSecurityClusterInstallationFlags(cmd)

	cmd.AddCommand(securityClusterInstallationLockCmd())
	cmd.AddCommand(securityClusterInstallationUnlockCmd())

	return cmd
}

func securityClusterInstallationLockCmd() *cobra.Command {

	var flags securityClusterInstallationFlags

	cmd := &cobra.Command{
		Use:   "api-lock",
		Short: "Lock API changes on a given cluster installation",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			if err := client.LockAPIForClusterInstallation(flags.clusterInstallationID); err != nil {
				return errors.Wrap(err, "failed to lock cluster installation API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.addFlags(cmd)
			return
		},
	}

	return cmd
}

func securityClusterInstallationUnlockCmd() *cobra.Command {

	var flags securityClusterInstallationFlags

	cmd := &cobra.Command{
		Use:   "api-unlock",
		Short: "Unlock API changes on a given cluster installation",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			if err := client.UnlockAPIForClusterInstallation(flags.clusterInstallationID); err != nil {
				return errors.Wrap(err, "failed to unlock cluster installation API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.addFlags(cmd)
			return
		},
	}

	return cmd
}

func securityGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "group",
		Short: "Manage security locks for group resources.",
	}

	setSecurityGroupFlags(cmd)

	cmd.AddCommand(securityGroupLockCmd())
	cmd.AddCommand(securityGroupUnlockCmd())

	return cmd
}

func securityGroupLockCmd() *cobra.Command {

	var flags securityGroupFlags

	cmd := &cobra.Command{
		Use:   "api-lock",
		Short: "Lock API changes on a given group",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			if err := client.LockAPIForGroup(flags.groupID); err != nil {
				return errors.Wrap(err, "failed to lock group API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.addFlags(cmd)
			return
		},
	}

	return cmd
}

func securityGroupUnlockCmd() *cobra.Command {

	var flags securityGroupFlags

	cmd := &cobra.Command{
		Use:   "api-unlock",
		Short: "Unlock API changes on a given group",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			if err := client.UnlockAPIForGroup(flags.groupID); err != nil {
				return errors.Wrap(err, "failed to unlock group API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.addFlags(cmd)
			return
		},
	}

	return cmd
}

func securityBackupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "backup",
		Short: "Manage security locks for backup resources.",
	}

	setSecurityBackupFlags(cmd)

	cmd.AddCommand(securityBackupLockCmd())
	cmd.AddCommand(securityBackupUnlockCmd())

	return cmd
}

func securityBackupLockCmd() *cobra.Command {

	var flags securityBackupFlags

	cmd := &cobra.Command{
		Use:   "api-lock",
		Short: "Lock API changes on a given backup",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			if err := client.LockAPIForBackup(flags.backupID); err != nil {
				return errors.Wrap(err, "failed to lock backup API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.addFlags(cmd)
			return
		},
	}

	return cmd
}

func securityBackupUnlockCmd() *cobra.Command {

	var flags securityBackupFlags

	cmd := &cobra.Command{
		Use:   "api-unlock",
		Short: "Unlock API changes on a given backup",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			if err := client.UnlockAPIForBackup(flags.backupID); err != nil {
				return errors.Wrap(err, "failed to unlock backup API")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.addFlags(cmd)
			return
		},
	}

	return cmd
}
