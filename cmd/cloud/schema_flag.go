// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
// 

package main

import "github.com/spf13/cobra"

func setSchemaFlags(command *cobra.Command) {
	command.PersistentFlags().String("database", "", "The database backing the provisioning server.")
}

type schemaFlag struct {
	database string
}

func (flags *schemaFlag) addFlags(command *cobra.Command) {
	flags.database, _ = command.Flags().GetString("database")
}
