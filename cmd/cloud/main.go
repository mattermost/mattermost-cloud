// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

// Package main is the entry point to the Mattermost Cloud provisioning server and CLI.
package main

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "cloud",
	Short: "Cloud is a tool to provision, manage, and monitor Kubernetes clusters.",
	Run: func(cmd *cobra.Command, args []string) {
		serverCmd.RunE(cmd, args)
	},
	// SilenceErrors allows us to explicitly log the error returned from rootCmd below.
	SilenceErrors: true,
}

func init() {
	rootCmd.MarkFlagRequired("database")

	rootCmd.AddCommand(serverCmd)
	rootCmd.AddCommand(clusterCmd)
	rootCmd.AddCommand(installationCmd)
	rootCmd.AddCommand(groupCmd)
	rootCmd.AddCommand(databaseCmd)
	rootCmd.AddCommand(schemaCmd)
	rootCmd.AddCommand(webhookCmd)
	rootCmd.AddCommand(securityCmd)
	rootCmd.AddCommand(workbenchCmd)
	rootCmd.AddCommand(completionCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Error("command failed")
		os.Exit(1)
	}
}
