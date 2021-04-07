// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	databaseCmd.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")

	databaseListCmd.Flags().String("vpc-id", "", "The VPC ID by which to filter databases.")
	databaseListCmd.Flags().String("database-type", "", "The database type by which to filter databases.")
	registerPagingFlags(databaseListCmd)

	databaseCmd.AddCommand(databaseListCmd)
}

var databaseCmd = &cobra.Command{
	Use:   "database",
	Short: "View information on known external multitenant databases",
}

var databaseListCmd = &cobra.Command{
	Use:   "list",
	Short: "List multitenant databases that are currently in use.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		vpcID, _ := command.Flags().GetString("vpc-id")
		databaseType, _ := command.Flags().GetString("database-type")
		paging := parsePagingFlags(command)

		databases, err := client.GetMultitenantDatabases(&model.GetDatabasesRequest{
			VpcID:        vpcID,
			DatabaseType: databaseType,
			Paging:       paging,
		})
		if err != nil {
			return errors.Wrap(err, "failed to query databases")
		}

		err = printJSON(databases)
		if err != nil {
			return err
		}

		return nil
	},
}
