package main

import "github.com/spf13/cobra"

type dashboardFlags struct {
	clusterFlags
	serverAddress  string
	refreshSeconds int
}

func (flags *dashboardFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.serverAddress, "server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
	command.Flags().IntVar(&flags.refreshSeconds, "refresh-seconds", 10, "The amount of seconds before the dashboard is refreshed with new data.")
}
