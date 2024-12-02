package main

import (
	"fmt"

	"github.com/mattermost/mattermost-cloud/cmd/cloud/clicontext"
	"github.com/mattermost/mattermost-cloud/internal/auth"
	"github.com/spf13/cobra"
)

type loginFlags struct {
	Context string
}

func (flags *loginFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.Context, "context", "", "The name of the context to use.")
}

func newCmdLogin() *cobra.Command {
	var flags loginFlags
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Login to the Mattermost Cloud provisioning server.",
		RunE: func(command *cobra.Command, args []string) error {
			contextName := flags.Context

			contexts, err := clicontext.ReadContexts()
			if err != nil {
				return err
			}

			if contextName == "" {
				contextName = contexts.CurrentContext
			}

			if _, exists := contexts.Contexts[contextName]; !exists {
				return fmt.Errorf("context '%s' does not exist", contextName)
			}

			context := contexts.Contexts[contextName]

			if context.ClientID == "" || context.OrgURL == "" {
				return fmt.Errorf("context '%s' has missing authentication data", contextName)
			}

			command.SilenceUsage = true

			login, err := auth.Login(command.Context(), context.OrgURL, context.ClientID)
			if err != nil {
				return err
			}

			context.AuthData = &login

			err = contexts.UpdateContext(contextName, context.AuthData, context.ClientID, context.OrgURL, context.Alias, context.ServerURL, context.ConfirmationRequired)
			return err
		},
	}

	flags.addFlags(cmd)

	return cmd
}
