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
	databaseListCmd.Flags().Int("page", 0, "The page of databases to fetch, starting at 0.")
	databaseListCmd.Flags().Int("per-page", 100, "The number of databases to fetch per page.")

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
		page, _ := command.Flags().GetInt("page")
		perPage, _ := command.Flags().GetInt("per-page")
		databases, err := client.GetMultitenantDatabases(&model.GetDatabasesRequest{
			VpcID:        vpcID,
			DatabaseType: databaseType,
			Page:         page,
			PerPage:      perPage,
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
