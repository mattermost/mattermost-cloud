package main

import "github.com/spf13/cobra"

type dashboardFlags struct {
	serverAddress  string
	refreshSeconds int
}

func (flags *dashboardFlags) addFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&flags.serverAddress, "server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
	cmd.PersistentFlags().IntVar(&flags.refreshSeconds, "refresh-seconds", 10, "The amount of seconds before the dashboard is refreshed with new data.")
}
