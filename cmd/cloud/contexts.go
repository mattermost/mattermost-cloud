package main

import (
	"fmt"
	"reflect"

	"github.com/mattermost/mattermost-cloud/cmd/cloud/clicontext"
	"github.com/mattermost/mattermost-cloud/internal/auth"
	"github.com/spf13/cobra"
)

func newCmdContexts() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contexts",
		Short: "Manipulate local contexts for the Cloud CLI",
	}

	setClusterFlags(cmd)

	cmd.AddCommand(newCmdContextList())
	cmd.AddCommand(newCmdContextGet())
	cmd.AddCommand(newCmdContextCreate())
	cmd.AddCommand(newCmdContextDelete())
	cmd.AddCommand(newCmdContextSet())
	cmd.AddCommand(newCmdContextUpdate())

	return cmd
}

func newCmdContextList() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all contexts.",
		RunE: func(cmd *cobra.Command, args []string) error {
			contexts, err := clicontext.ReadContexts()
			if err != nil {
				return err
			}

			for name, ctx := range contexts.Contexts {
				current := ""
				if name == contexts.CurrentContext {
					current = " (current)"
				}
				fmt.Printf("%s%s\n", name, current)
				fmt.Printf("  Server URL: %s\n", ctx.ServerURL)
				fmt.Printf("  Client ID: %s\n", ctx.ClientID)
				fmt.Printf("  Org URL: %s\n", ctx.OrgURL)
			}

			return nil
		},
	}
}

func newCmdContextGet() *cobra.Command {
	return &cobra.Command{
		Use:   "get <context-name>",
		Short: "Get details of a specific context.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			contextName := args[0]
			contexts, err := clicontext.ReadContexts()
			if err != nil {
				return err
			}

			ctx, ok := contexts.Contexts[contextName]
			if !ok {
				return fmt.Errorf("context '%s' not found", contextName)
			}

			fmt.Printf("Name: %s\n", contextName)
			fmt.Printf("  Server URL: %s\n", ctx.ServerURL)
			fmt.Printf("  Client ID: %s\n", ctx.ClientID)
			fmt.Printf("  Org URL: %s\n", ctx.OrgURL)

			return nil
		},
	}
}

type updateContextFlags struct {
	createContextFlags
	Context   string
	ClearAuth bool
}

func (f *updateContextFlags) addFlags(command *cobra.Command) {
	f.createContextFlags.addFlags(command)
	command.Flags().StringVar(&f.Context, "context", "", "Name of the context to update")
	command.MarkFlagRequired("context")
	command.Flags().BoolVar(&f.ClearAuth, "clear-auth", false, "Turns off authentication for this context")
}

func newCmdContextUpdate() *cobra.Command {
	var flags updateContextFlags
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a context.",
		RunE: func(command *cobra.Command, args []string) error {

			serverAddress := flags.ServerURL
			clientID := flags.ClientID
			orgURL := flags.OrgURL
			skipAuth := flags.SkipAuth
			Alias := flags.Alias
			clearAuth := flags.ClearAuth
			contextName := flags.Context

			contexts, err := clicontext.ReadContexts()
			if err != nil {
				return err
			}

			if _, exists := contexts.Contexts[contextName]; !exists {
				return fmt.Errorf("context '%s' not found", contextName)
			}

			context := contexts.Contexts[contextName]

			if serverAddress != "" && context.ServerURL != serverAddress {
				context.ServerURL = serverAddress
			}

			if clientID != "" && context.ClientID != clientID {
				context.ClientID = clientID
			}

			if orgURL != "" && context.OrgURL != orgURL {
				context.OrgURL = orgURL
			}

			if Alias != "" && context.Alias != Alias {
				context.Alias = Alias
			}

			if !skipAuth && !clearAuth && clientID != "" && orgURL != "" {
				login, err := auth.Login(command.Context(), orgURL, clientID)
				if err != nil {
					return err
				}

				context.AuthData = &login
			}

			if clearAuth {
				context.AuthData = nil
				context.ClientID = ""
				context.OrgURL = ""
			}

			contexts.Contexts[contextName] = context

			return clicontext.WriteContexts(contexts)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

type createContextFlags struct {
	clusterFlags
	ClientID  string
	OrgURL    string
	SkipAuth  bool
	Alias     string
	ServerURL string
}

func (f *createContextFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&f.ClientID, "client-id", "", "OAuth2 Client ID for the context")
	command.Flags().StringVar(&f.OrgURL, "org-url", "", "Organization URL for the context")
	command.Flags().BoolVar(&f.SkipAuth, "skip-auth", false, "Skips kicking off authentication for this context")
	command.Flags().StringVar(&f.Alias, "alias", "", "Alias for the context, defaults to server URL")
	command.Flags().StringVar(&f.ServerURL, "server-url", "", "Server URL for the context")
}

func newCmdContextCreate() *cobra.Command {
	var flags createContextFlags
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a new context.",
		RunE: func(command *cobra.Command, args []string) error {
			serverAddress := flags.ServerURL
			clientID := flags.ClientID
			orgURL := flags.OrgURL
			skipAuth := flags.SkipAuth
			Alias := flags.Alias
			contextName := ""

			if Alias != "" {
				contextName = Alias
			} else {
				contextName = serverAddress
			}

			contexts, err := clicontext.ReadContexts()
			if err != nil {
				return err
			}

			if _, exists := contexts.Contexts[contextName]; exists {
				return fmt.Errorf("context '%s' already exists", contextName)
			}

			var authData *auth.AuthorizationResponse
			if !skipAuth && clientID != "" && orgURL != "" {
				login, err := auth.Login(command.Context(), orgURL, clientID)
				if err != nil {
					return err
				}

				authData = &login
			}

			contexts.Contexts[contextName] = clicontext.CLIContext{
				ClientID:  clientID,
				OrgURL:    orgURL,
				AuthData:  authData,
				Alias:     Alias,
				ServerURL: serverAddress,
			}

			contexts.CurrentContext = contextName

			return clicontext.WriteContexts(contexts)
		},
		PreRun: func(command *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(command)
		},
	}

	flags.addFlags(cmd)
	cmd.MarkFlagRequired("server-url")

	return cmd
}

type deleteContextFlags struct {
	clusterFlags
	contextName string
}

func (f *deleteContextFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&f.contextName, "context", "", "Name of the context to delete")
	command.MarkFlagRequired("context")
}

func newCmdContextDelete() *cobra.Command {
	var flags deleteContextFlags
	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a context.",
		RunE: func(command *cobra.Command, args []string) error {
			contextName := flags.contextName

			contexts, err := clicontext.ReadContexts()
			if err != nil {
				return err
			}

			if _, exists := contexts.Contexts[contextName]; !exists {
				return fmt.Errorf("context '%s' not found", contextName)
			}

			delete(contexts.Contexts, contextName)

			if len(contexts.Contexts) == 0 {
				contexts.CurrentContext = ""
			} else {
				// Set the Current Context to the first context in the map
				contexts.CurrentContext = reflect.ValueOf(contexts.Contexts).MapKeys()[0].String()
			}

			return clicontext.WriteContexts(contexts)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdContextSet() *cobra.Command {
	return &cobra.Command{
		Use:   "set-current <context-name>",
		Short: "Set the current context.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			contextName := args[0]

			contexts, err := clicontext.ReadContexts()
			if err != nil {
				return err
			}

			if _, exists := contexts.Contexts[contextName]; !exists {
				return fmt.Errorf("context '%s' not found", contextName)
			}

			contexts.CurrentContext = contextName

			return clicontext.WriteContexts(contexts)
		},
	}
}
