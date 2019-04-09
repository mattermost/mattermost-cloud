package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var groupCmd = &cobra.Command{
	Use:   "group",
	Short: "Manipulate groups managed by the provisioning server.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("TODO: implement group command")
	},
}
