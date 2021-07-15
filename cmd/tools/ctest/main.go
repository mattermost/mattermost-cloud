// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

// Package main is the entry point to the Mattermost Cloud provisioning server and CLI.
package main

import (
	"os"

	"github.com/spf13/cobra"
)

const (
	defaultListenerPort      = "8085"
	defaultLocalServerAPI    = "http://localhost:8075"
	defaultMattermostVersion = "5.36.1"
)

var rootCmd = &cobra.Command{
	Use:   "ctest",
	Short: "A testing suite tool for the cloud server",
	// SilenceErrors allows us to explicitly log the error returned from rootCmd below.
	SilenceErrors: true,
}

func init() {
	// General Settings
	rootCmd.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
	rootCmd.PersistentFlags().String("webhook-listener-port", defaultListenerPort, "The local webhook listening port.")
	rootCmd.PersistentFlags().String("webhook-url", "http://localhost:8085", "The listener URL of this tool which can be reached from the provisioner. (hint: use ngrok when not testing local provisioners)")

	// Installation Settings
	rootCmd.PersistentFlags().String("version", defaultMattermostVersion, "The Mattermost version to install.")
	rootCmd.PersistentFlags().String("license", "", "The license to use in the installation.")
	rootCmd.PersistentFlags().String("installation-domain", "dev.cloud.mattermost.com", "The domain under which the test installations will be created.")

	rootCmd.AddCommand(databaseCmd)
	rootCmd.AddCommand(filestoreCmd)
	rootCmd.AddCommand(generateCmd)
	rootCmd.AddCommand(deleteCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Error("command failed")
		os.Exit(1)
	}
}
