package main

import (
	"strings"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	clusterInstallationCmd.PersistentFlags().String("server", "http://localhost:8075", "The provisioning server whose API will be queried.")

	clusterInstallationGetCmd.Flags().String("cluster-installation", "", "The id of the cluster installation to be fetched.")
	clusterInstallationGetCmd.MarkFlagRequired("cluster-installation")

	clusterInstallationListCmd.Flags().String("cluster", "", "The cluster by which to filter cluster installations.")
	clusterInstallationListCmd.Flags().String("installation", "", "The installation by which to filter cluster installations.")
	clusterInstallationListCmd.Flags().Int("page", 0, "The page of cluster installations to fetch, starting at 0.")
	clusterInstallationListCmd.Flags().Int("per-page", 100, "The number of cluster installations to fetch per page.")
	clusterInstallationListCmd.Flags().Bool("include-deleted", false, "Whether to include deleted cluster installations.")

	clusterInstallationConfigCmd.PersistentFlags().String("cluster-installation", "", "The id of the cluster installation.")
	clusterInstallationConfigCmd.MarkFlagRequired("cluster-installation")

	clusterInstallationConfigSetCmd.Flags().String("key", "", "The configuration key to update (e.g. ServiceSettings.SiteURL).")
	clusterInstallationConfigSetCmd.Flags().String("value", "", "The value to write to the config.")
	clusterInstallationConfigSetCmd.MarkFlagRequired("key")
	clusterInstallationConfigSetCmd.MarkFlagRequired("value")

	clusterInstallationCmd.AddCommand(clusterInstallationGetCmd)
	clusterInstallationCmd.AddCommand(clusterInstallationListCmd)
	clusterInstallationCmd.AddCommand(clusterInstallationConfigCmd)

	clusterInstallationConfigCmd.AddCommand(clusterInstallationConfigGetCmd)
	clusterInstallationConfigCmd.AddCommand(clusterInstallationConfigSetCmd)
}

var clusterInstallationCmd = &cobra.Command{
	Use:   "installation",
	Short: "Manipulate cluster installations managed by the provisioning server.",
}

var clusterInstallationGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a particular cluster installation.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterInstallationID, _ := command.Flags().GetString("cluster-installation")
		clusterInstallation, err := client.GetClusterInstallation(clusterInstallationID)
		if err != nil {
			return errors.Wrap(err, "failed to query cluster installation")
		}
		if clusterInstallation == nil {
			return nil
		}

		err = printJSON(clusterInstallation)
		if err != nil {
			return err
		}

		return nil
	},
}

var clusterInstallationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List created cluster installations.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		cluster, _ := command.Flags().GetString("cluster")
		installation, _ := command.Flags().GetString("installation")
		page, _ := command.Flags().GetInt("page")
		perPage, _ := command.Flags().GetInt("per-page")
		includeDeleted, _ := command.Flags().GetBool("include-deleted")

		clusterInstallations, err := client.GetClusterInstallations(&model.GetClusterInstallationsRequest{
			ClusterID:      cluster,
			InstallationID: installation,
			Page:           page,
			PerPage:        perPage,
			IncludeDeleted: includeDeleted,
		})
		if err != nil {
			return errors.Wrap(err, "failed to query cluster installations")
		}

		err = printJSON(clusterInstallations)
		if err != nil {
			return err
		}

		return nil
	},
}

var clusterInstallationConfigCmd = &cobra.Command{
	Use:   "config",
	Short: "Manipulate a particular cluster installation's config.",
}

var clusterInstallationConfigGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a particular cluster installation's config.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterInstallationID, _ := command.Flags().GetString("cluster-installation")
		clusterInstallationConfig, err := client.GetClusterInstallationConfig(clusterInstallationID)
		if err != nil {
			return errors.Wrap(err, "failed to query cluster installation config")
		}
		if clusterInstallationConfig == nil {
			return nil
		}

		err = printJSON(clusterInstallationConfig)
		if err != nil {
			return err
		}

		return nil
	},
}

var clusterInstallationConfigSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set a particular cluster installation's config.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterInstallationID, _ := command.Flags().GetString("cluster-installation")
		key, _ := command.Flags().GetString("key")
		value, _ := command.Flags().GetString("value")

		config := make(map[string]interface{})
		keyParts := strings.Split(key, ".")
		configRef := config
		for i, keyPart := range keyParts {
			if i < len(keyParts)-1 {
				value := make(map[string]interface{})
				configRef[keyPart] = value
				configRef = value
			} else {
				configRef[keyPart] = value
			}
		}

		err := client.SetClusterInstallationConfig(clusterInstallationID, config)
		if err != nil {
			return errors.Wrap(err, "failed to modify cluster installation config")
		}

		return nil
	},
}
