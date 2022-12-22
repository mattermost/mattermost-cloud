package main

import "github.com/spf13/cobra"

func setSchemaFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("database", "sqlite://cloud.db", "The database backing the provisioning server.")
}

type schemaFlag struct {
	database string
}

func (flags *schemaFlag) addFlags(cmd *cobra.Command) {
	flags.database, _ = cmd.Flags().GetString("database")
}
