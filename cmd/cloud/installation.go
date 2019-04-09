package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var installationCmd = &cobra.Command{
	Use:   "installation",
	Short: "Manipulate installations managed by the provisioning server.",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("TODO: implement installation command")
	},
}
