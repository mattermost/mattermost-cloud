// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

// Package main is the entry point to the Mattermost Cloud provisioning server and CLI.
package main

import (
	"context"
	"os"
	"strings"

	_ "github.com/golang/mock/mockgen/model"
	"github.com/mattermost/mattermost-cloud/cmd/cloud/clicontext"
	"github.com/mattermost/mattermost-cloud/internal/auth"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

var instanceID string

func isCommandThatSkipsAuth(cmd *cobra.Command) bool {
	return cmd.Use == "migrate" || cmd.Use == "server" || cmd.Use == "login" || cmd.Parent().Use == "contexts"
}

// TODO: Add support for --context flag to all commands that can use a context different from current one
var rootCmd = &cobra.Command{
	Use:   "cloud",
	Short: "Cloud is a tool to provision, manage, and monitor Kubernetes clusters.",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		populateEnv(cmd)
		if !isCommandThatSkipsAuth(cmd) {
			contexts, err := clicontext.ReadContexts()
			if err != nil {
				logger.WithError(err).Fatal("Failed to read contexts from disk.")
				return
			}

			contextOverride, _ := cmd.Flags().GetString("context")

			var currentContext *clicontext.CLIContext
			if contextOverride != "" {
				currentContext = contexts.Get(contextOverride)
				if currentContext == nil {
					logger.Fatalf("Context '%s' does not exist.", contextOverride)
					return
				}
			} else {
				currentContext = contexts.Current()
			}

			if currentContext == nil {
				logger.Fatal("No current context set. Use 'cloud contexts set' to set the current context.")
				return
			}

			// If there's no Client ID configured, don't mess with authentication
			var authData *auth.AuthorizationResponse
			if currentContext.ClientID != "" {
				authData, err = auth.EnsureValidAuthData(cmd.Context(), currentContext.AuthData, currentContext.OrgURL, currentContext.ClientID)
				if err != nil {
					logger.WithError(err).Fatal("Failed to ensure valid authentication data.")
					return
				}

				if authData == nil {
					logger.Fatal("Failed to authenticate.")
					return
				}
				authData.ExpiresAt = authData.GetExpiresAt()

				cmd.SetContext(context.WithValue(cmd.Context(), auth.ContextKeyAuthData{}, authData))
			}

			// Update disk copy of context with new auth data if any
			err = contexts.UpdateContext(currentContext.Alias, authData, currentContext.ClientID, currentContext.OrgURL, currentContext.Alias, currentContext.ServerURL)
			if err != nil {
				logger.WithError(err).Fatal("Failed to update context with new auth data.")
			}
			cmd.SetContext(context.WithValue(cmd.Context(), clicontext.ContextKeyServerURL{}, currentContext.ServerURL))
		}
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		return newCmdServer().RunE(cmd, args)
	},
	// SilenceErrors allows us to explicitly log the error returned from rootCmd below.
	SilenceErrors: true,
}

func init() {
	// If the environment variable "HOSTNAME" is set then that will be used for
	// the instance ID value of this server. In Kubernetes this should pick up
	// the pod replica name. If this environment value is not set then the
	// original behavior will kick in and set the ID to a random ID.
	instanceID = os.Getenv("HOSTNAME")
	if len(instanceID) == 0 {
		instanceID = model.NewID()
	}

	_ = rootCmd.MarkFlagRequired("database")

	rootCmd.PersistentFlags().String("context", viper.GetString("context"), "Override the current context")

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
	rootCmd.AddCommand(newCmdLogin())
	rootCmd.AddCommand(newCmdContexts())
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
