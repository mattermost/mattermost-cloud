package main

import "github.com/spf13/cobra"

func getStringFlagPointer(command *cobra.Command, s string) *string {
	if command.Flags().Changed(s) {
		val, _ := command.Flags().GetString(s)
		return &val
	}

	return nil
}

func getInt64FlagPointer(command *cobra.Command, s string) *int64 {
	if command.Flags().Changed(s) {
		val, _ := command.Flags().GetInt64(s)
		return &val
	}

	return nil
}
