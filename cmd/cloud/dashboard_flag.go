package main

import "github.com/spf13/cobra"

type dashboardFlags struct {
	clusterFlags
	refreshSeconds int
}

func (flags *dashboardFlags) addFlags(command *cobra.Command) {
	command.Flags().IntVar(&flags.refreshSeconds, "refresh-seconds", 10, "The amount of seconds before the dashboard is refreshed with new data.")
}
