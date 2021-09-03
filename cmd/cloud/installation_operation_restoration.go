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
	installationRestorationRequestCmd.Flags().String("installation", "", "The id of the installation to be restored.")
	installationRestorationRequestCmd.Flags().String("backup", "", "The id of the backup to restore.")
	installationRestorationRequestCmd.MarkFlagRequired("installation")
	installationRestorationRequestCmd.MarkFlagRequired("backup")

	installationRestorationsListCmd.Flags().String("installation", "", "The id of the installation to query operations.")
	installationRestorationsListCmd.Flags().String("state", "", "The state to filter operations by.")
	installationRestorationsListCmd.Flags().String("cluster-installation", "", "The cluster installation to filter operations by.")
	registerTableOutputFlags(installationRestorationsListCmd)
	registerPagingFlags(installationRestorationsListCmd)

	installationRestorationGetCmd.Flags().String("restoration", "", "The id of restoration operation.")
	installationRestorationGetCmd.MarkFlagRequired("restoration")

	installationRestorationOperationCmd.AddCommand(installationRestorationRequestCmd)
	installationRestorationOperationCmd.AddCommand(installationRestorationsListCmd)
	installationRestorationOperationCmd.AddCommand(installationRestorationGetCmd)
}

var installationRestorationOperationCmd = &cobra.Command{
	Use:   "restoration",
	Short: "Manipulate installation restoration operations managed by the provisioning server.",
}

var installationRestorationRequestCmd = &cobra.Command{
	Use:   "request",
	Short: "Request database restoration",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		backupID, _ := command.Flags().GetString("backup")

		installationDTO, err := client.RestoreInstallationDatabase(installationID, backupID)
		if err != nil {
			return errors.Wrap(err, "failed to request installation database restoration")
		}

		err = printJSON(installationDTO)
		if err != nil {
			return err
		}

		return nil
	},
}

var installationRestorationsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installation database restoration operations",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		clusterInstallationID, _ := command.Flags().GetString("cluster-installation")
		state, _ := command.Flags().GetString("state")
		paging := parsePagingFlags(command)

		request := &model.GetInstallationDBRestorationOperationsRequest{
			Paging:                paging,
			InstallationID:        installationID,
			ClusterInstallationID: clusterInstallationID,
			State:                 state,
		}

		dbRestorationOperations, err := client.GetInstallationDBRestorationOperations(request)
		if err != nil {
			return errors.Wrap(err, "failed to list installation database restoration operations")
		}

		if enabled, customCols := tableOutputEnabled(command); enabled {
			var keys []string
			var vals [][]string

			if len(customCols) > 0 {
				data := make([]interface{}, 0, len(dbRestorationOperations))
				for _, elem := range dbRestorationOperations {
					data = append(data, elem)
				}
				keys, vals, err = prepareTableData(customCols, data)
				if err != nil {
					return errors.Wrap(err, "failed to prepare table output")
				}
			} else {
				keys, vals = defaultDBRestorationOperationTableData(dbRestorationOperations)
			}

			printTable(keys, vals)
			return nil
		}

		err = printJSON(dbRestorationOperations)
		if err != nil {
			return err
		}

		return nil
	},
}

func defaultDBRestorationOperationTableData(ops []*model.InstallationDBRestorationOperation) ([]string, [][]string) {
	keys := []string{"ID", "INSTALLATION ID", "BACKUP ID", "STATE", "CLUSTER INSTALLATION ID", "TARGET INSTALLATION STATE", "REQUEST AT"}
	vals := make([][]string, 0, len(ops))

	for _, restoration := range ops {
		vals = append(vals, []string{
			restoration.ID,
			restoration.InstallationID,
			restoration.BackupID,
			string(restoration.State),
			restoration.ClusterInstallationID,
			restoration.TargetInstallationState,
			model.TimeFromMillis(restoration.RequestAt).Format("2006-01-02 15:04:05 -0700 MST"),
		})
	}
	return keys, vals
}

var installationRestorationGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Fetches given installation database restoration operation.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		restorationID, _ := command.Flags().GetString("restoration")

		restorationOperation, err := client.GetInstallationDBRestoration(restorationID)
		if err != nil {
			return errors.Wrap(err, "failed to get installation database restoration")
		}

		err = printJSON(restorationOperation)
		if err != nil {
			return err
		}

		return nil
	},
}
