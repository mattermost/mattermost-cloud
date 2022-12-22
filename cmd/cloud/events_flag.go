package main

import "github.com/spf13/cobra"

func setEventFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
}

type eventFlags struct {
	serverAddress string
}

func (flags *eventFlags) addFlags(cmd *cobra.Command) {
	flags.serverAddress, _ = cmd.Flags().GetString("server")
}

type stateChangeEventListFlags struct {
	eventFlags
	pagingFlags
	tableOptions
	resourceType string
	resourceID   string
}

func (flags *stateChangeEventListFlags) addFlags(cmd *cobra.Command) {
	flags.pagingFlags.addFlags(cmd)
	flags.tableOptions.addFlags(cmd)
	cmd.Flags().StringVar(&flags.resourceType, "resource-type", "", "Type of a resource for which to list events.")
	cmd.Flags().StringVar(&flags.resourceID, "resource-id", "", "ID of a resource for which to list events.")
}
