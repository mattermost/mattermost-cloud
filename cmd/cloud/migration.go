package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	migrationCmd.PersistentFlags().String("server", "http://localhost:8075", "The provisioning server whose API will be queried.")

	migrationCreateCmd.Flags().String("cluster-id", "", "ID of the cluster where the installation will be migrated to.")
	migrationCreateCmd.Flags().String("cluster-installation-id", "", "ID of the cluster installation to be migrated from.")
	migrationCreateCmd.MarkFlagRequired("cluster-id")
	migrationCreateCmd.MarkFlagRequired("cluster-installation-id")

	migrationCmd.AddCommand(migrationCreateCmd)
}

var migrationCmd = &cobra.Command{
	Use:   "migration",
	Short: "Manipulate migrations managed by the provisioning server.",
}

var migrationCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an migration.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterID, _ := command.Flags().GetString("cluster-id")
		clusterInstallationID, _ := command.Flags().GetString("cluster-installation-id")

		migration, err := client.CreateClusterInstallationMigration(&model.CreateClusterInstallationMigrationRequest{
			ClusterID:             clusterID,
			ClusterInstallationID: clusterInstallationID,
		})
		if err != nil {
			return errors.Wrap(err, "failed to create migration")
		}

		err = printJSON(migration)
		if err != nil {
			return err
		}

		return nil
	},
}
