// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

// Package main is the entry point to the Mattermost Cloud provisioning server and CLI.
package main

import (
	"os"
	"strings"

	_ "github.com/golang/mock/mockgen/model"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var instanceID string

var rootCmd = &cobra.Command{
	Use:   "cloud",
	Short: "Cloud is a tool to provision, manage, and monitor Kubernetes clusters.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		populateEnv(cmd)
	},
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

func populateEnv(cmd *cobra.Command) {
	v := viper.New()

	v.SetEnvPrefix("cp")
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	v.AutomaticEnv()

	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		if !f.Changed {
			if v.IsSet(f.Name) {
				_ = cmd.Flags().Set(f.Name, v.GetString(f.Name))
			}
		}
	})
}
