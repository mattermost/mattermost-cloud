package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Run the provisioning server.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("TODO: implement server command")
	},
}
