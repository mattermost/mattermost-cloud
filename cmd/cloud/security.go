// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	securityCmd.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")

	securityClusterCmd.PersistentFlags().String("cluster", "", "The id of the cluster.")
	securityClusterCmd.MarkPersistentFlagRequired("cluster")

	securityInstallationCmd.PersistentFlags().String("installation", "", "The id of the installation.")
	securityInstallationCmd.MarkPersistentFlagRequired("installation")

	securityClusterInstallationCmd.PersistentFlags().String("cluster-installation", "", "The id of the cluster installation.")
	securityClusterInstallationCmd.MarkPersistentFlagRequired("cluster-installation")

	securityGroupCmd.PersistentFlags().String("group", "", "The id of the group.")
	securityGroupCmd.MarkPersistentFlagRequired("group")

	securityCmd.AddCommand(securityClusterCmd)
	securityClusterCmd.AddCommand(securityClusterLockAPICmd)
	securityClusterCmd.AddCommand(securityClusterUnlockAPICmd)

	securityCmd.AddCommand(securityInstallationCmd)
	securityInstallationCmd.AddCommand(securityInstallationLockAPICmd)
	securityInstallationCmd.AddCommand(securityInstallationUnlockAPICmd)

	securityCmd.AddCommand(securityClusterInstallationCmd)
	securityClusterInstallationCmd.AddCommand(securityClusterInstallationLockAPICmd)
	securityClusterInstallationCmd.AddCommand(securityClusterInstallationUnlockAPICmd)

	securityCmd.AddCommand(securityGroupCmd)
	securityGroupCmd.AddCommand(securityGroupLockAPICmd)
	securityGroupCmd.AddCommand(securityGroupUnlockAPICmd)
}

var securityCmd = &cobra.Command{
	Use:   "security",
	Short: "Manage security locks for different cloud resources.",
}

var securityClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manage security locks for cluster resources.",
}

var securityClusterLockAPICmd = &cobra.Command{
	Use:   "api-lock",
	Short: "Lock API changes on a given cluster",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterID, _ := command.Flags().GetString("cluster")
		err := client.LockAPIForCluster(clusterID)
		if err != nil {
			return errors.Wrap(err, "failed to lock cluster API")
		}

		return nil
	},
}

var securityClusterUnlockAPICmd = &cobra.Command{
	Use:   "api-unlock",
	Short: "Unlock API changes on a given cluster",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterID, _ := command.Flags().GetString("cluster")
		err := client.UnlockAPIForCluster(clusterID)
		if err != nil {
			return errors.Wrap(err, "failed to unlock cluster API")
		}

		return nil
	},
}

var securityInstallationCmd = &cobra.Command{
	Use:   "installation",
	Short: "Manage security locks for installation resources.",
}

var securityInstallationLockAPICmd = &cobra.Command{
	Use:   "api-lock",
	Short: "Lock API changes on a given installation",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		err := client.LockAPIForInstallation(installationID)
		if err != nil {
			return errors.Wrap(err, "failed to lock installation API")
		}

		return nil
	},
}

var securityInstallationUnlockAPICmd = &cobra.Command{
	Use:   "api-unlock",
	Short: "Unlock API changes on a given installation",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		err := client.UnlockAPIForInstallation(installationID)
		if err != nil {
			return errors.Wrap(err, "failed to unlock installation API")
		}

		return nil
	},
}

var securityClusterInstallationCmd = &cobra.Command{
	Use:   "cluster-installation",
	Short: "Manage security locks for cluster installation resources.",
}

var securityClusterInstallationLockAPICmd = &cobra.Command{
	Use:   "api-lock",
	Short: "Lock API changes on a given cluster installation",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterInstallationID, _ := command.Flags().GetString("cluster-installation")
		err := client.LockAPIForClusterInstallation(clusterInstallationID)
		if err != nil {
			return errors.Wrap(err, "failed to lock cluster installation API")
		}

		return nil
	},
}

var securityClusterInstallationUnlockAPICmd = &cobra.Command{
	Use:   "api-unlock",
	Short: "Unlock API changes on a given cluster installation",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterInstallationID, _ := command.Flags().GetString("cluster-installation")
		err := client.UnlockAPIForClusterInstallation(clusterInstallationID)
		if err != nil {
			return errors.Wrap(err, "failed to unlock cluster installation API")
		}

		return nil
	},
}

var securityGroupCmd = &cobra.Command{
	Use:   "group",
	Short: "Manage security locks for group resources.",
}

var securityGroupLockAPICmd = &cobra.Command{
	Use:   "api-lock",
	Short: "Lock API changes on a given group",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		groupID, _ := command.Flags().GetString("group")
		err := client.LockAPIForGroup(groupID)
		if err != nil {
			return errors.Wrap(err, "failed to lock group API")
		}

		return nil
	},
}

var securityGroupUnlockAPICmd = &cobra.Command{
	Use:   "api-unlock",
	Short: "Unlock API changes on a given group",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		groupID, _ := command.Flags().GetString("group")
		err := client.UnlockAPIForGroup(groupID)
		if err != nil {
			return errors.Wrap(err, "failed to unlock group API")
		}

		return nil
	},
}
