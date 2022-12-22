// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-cloud/internal/store"
)

func schemaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema",
		Short: "Manipulate the schema used by the provisioning server.",
	}

	setSchemaFlags(cmd)

	cmd.AddCommand(schemaMigrateCmd())

	return cmd
}

func sqlStore(database string) (*store.SQLStore, error) {
	sqlStore, err := store.New(database, logger)
	if err != nil {
		return nil, err
	}

	return sqlStore, nil
}

func schemaMigrateCmd() *cobra.Command {
	var flags schemaFlag
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate the schema to the latest supported version.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			sqlStore, err := sqlStore(flags.database)
			if err != nil {
				return err
			}

			return sqlStore.Migrate()
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.addFlags(cmd)
			return
		},
	}

	return cmd

}
