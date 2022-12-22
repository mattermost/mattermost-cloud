package main

import "github.com/spf13/cobra"

func setEventFlags(command *cobra.Command) {
	command.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
}

type eventFlags struct {
	serverAddress string
}

func (flags *eventFlags) addFlags(command *cobra.Command) {
	flags.serverAddress, _ = command.Flags().GetString("server")
}

type stateChangeEventListFlags struct {
	eventFlags
	pagingFlags
	tableOptions
	resourceType string
	resourceID   string
}

func (flags *stateChangeEventListFlags) addFlags(command *cobra.Command) {
	flags.pagingFlags.addFlags(command)
	flags.tableOptions.addFlags(command)

	command.Flags().StringVar(&flags.resourceType, "resource-type", "", "Type of a resource for which to list events.")
	command.Flags().StringVar(&flags.resourceID, "resource-id", "", "ID of a resource for which to list events.")
}
