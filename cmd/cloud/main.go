// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

// Package main is the entry point to the Mattermost Cloud provisioning server and CLI.
package main

import (
	"os"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/spf13/cobra"
)

var instanceID string

var rootCmd = &cobra.Command{
	Use:   "cloud",
	Short: "Cloud is a tool to provision, manage, and monitor Kubernetes clusters.",
	RunE: func(cmd *cobra.Command, args []string) error {
		return newCmdServer().RunE(cmd, args)
	},
	// SilenceErrors allows us to explicitly log the error returned from rootCmd below.
	SilenceErrors: true,
}

func init() {
	instanceID = model.NewID()

	_ = rootCmd.MarkFlagRequired("database")

	rootCmd.AddCommand(newCmdServer())
	rootCmd.AddCommand(newCmdCluster())
	rootCmd.AddCommand(newCmdInstallation())
	rootCmd.AddCommand(newCmdGroup())
	rootCmd.AddCommand(newCmdDatabase())
	rootCmd.AddCommand(newCmdSchema())
	rootCmd.AddCommand(newCmdWebhook())
	rootCmd.AddCommand(newCmdSecurity())
	rootCmd.AddCommand(newCmdWorkbench())
	rootCmd.AddCommand(newCmdCompletion())
	rootCmd.AddCommand(newCmdDashboard())
	rootCmd.AddCommand(newCmdEvents())
	rootCmd.AddCommand(newCmdSubscription())
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Error("command failed")
		os.Exit(1)
	}
}
