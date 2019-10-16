package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	installationCmd.PersistentFlags().String("server", "http://localhost:8075", "The provisioning server whose API will be queried.")

	installationCreateCmd.Flags().String("owner", "", "An opaque identifier describing the owner of the installation.")
	installationCreateCmd.Flags().String("version", "stable", "The Mattermost version to install.")
	installationCreateCmd.Flags().String("dns", "", "The URL at which the Mattermost server will be available.")
	installationCreateCmd.Flags().String("size", model.InstallationDefaultSize, "The size of the installation. Accepts 100users, 1000users, 5000users, 10000users, 25000users, miniSingleton, or miniHA. Defaults to 100users.")
	installationCreateCmd.Flags().String("affinity", model.InstallationAffinityIsolated, "How other installations may be co-located in the same cluster.")
	installationCreateCmd.Flags().String("license", "", "The Mattermost License to use in the server.")
	installationCreateCmd.Flags().String("database", model.InstallationDatabaseMysqlOperator, "The Mattermost server database type. Accepts mysql-operator or aws-rds")
	installationCreateCmd.Flags().String("filestore", model.InstallationFilestoreMinioOperator, "The Mattermost server filestore type. Accepts minio-operator or aws-s3")
	installationCreateCmd.MarkFlagRequired("owner")
	installationCreateCmd.MarkFlagRequired("dns")

	installationUpgradeCmd.Flags().String("installation", "", "The id of the installation to be upgraded.")
	installationUpgradeCmd.Flags().String("version", "stable", "The Mattermost version to target.")
	installationUpgradeCmd.Flags().String("license", "", "The Mattermost License to use in the server.")
	installationUpgradeCmd.MarkFlagRequired("installation")
	installationUpgradeCmd.MarkFlagRequired("version")

	installationDeleteCmd.Flags().String("installation", "", "The id of the installation to be deleted.")
	installationDeleteCmd.MarkFlagRequired("installation")

	installationGetCmd.Flags().String("installation", "", "The id of the installation to be fetched.")
	installationGetCmd.MarkFlagRequired("installation")

	installationListCmd.Flags().String("owner", "", "The owner by which to filter installations.")
	installationListCmd.Flags().Int("page", 0, "The page of installations to fetch, starting at 0.")
	installationListCmd.Flags().Int("per-page", 100, "The number of installations to fetch per page.")
	installationListCmd.Flags().Bool("include-deleted", false, "Whether to include deleted installations.")

	installationCmd.AddCommand(installationCreateCmd)
	installationCmd.AddCommand(installationUpgradeCmd)
	installationCmd.AddCommand(installationDeleteCmd)
	installationCmd.AddCommand(installationGetCmd)
	installationCmd.AddCommand(installationListCmd)
}

var installationCmd = &cobra.Command{
	Use:   "installation",
	Short: "Manipulate installations managed by the provisioning server.",
}

var installationCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create an installation.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		ownerID, _ := command.Flags().GetString("owner")
		version, _ := command.Flags().GetString("version")
		size, _ := command.Flags().GetString("size")
		dns, _ := command.Flags().GetString("dns")
		affinity, _ := command.Flags().GetString("affinity")
		license, _ := command.Flags().GetString("license")
		database, _ := command.Flags().GetString("database")
		filestore, _ := command.Flags().GetString("filestore")

		installation, err := client.CreateInstallation(&model.CreateInstallationRequest{
			OwnerID:   ownerID,
			Version:   version,
			Size:      size,
			DNS:       dns,
			License:   license,
			Affinity:  affinity,
			Database:  database,
			Filestore: filestore,
		})
		if err != nil {
			return errors.Wrap(err, "failed to create installation")
		}

		err = printJSON(installation)
		if err != nil {
			return err
		}

		return nil
	},
}

var installationUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade (or downgrade) the version of Mattermost.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		version, _ := command.Flags().GetString("version")
		license, _ := command.Flags().GetString("license")

		upgradeRequest := &model.UpgradeInstallationRequest{
			Version: version,
			License: license,
		}

		err := client.UpgradeInstallation(installationID, upgradeRequest)
		if err != nil {
			return errors.Wrap(err, "failed to change installation version")
		}

		return nil
	},
}

var installationDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete an installation.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")

		err := client.DeleteInstallation(installationID)
		if err != nil {
			return errors.Wrap(err, "failed to delete installation")
		}

		return nil
	},
}

var installationGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a particular installation.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		installation, err := client.GetInstallation(installationID)
		if err != nil {
			return errors.Wrap(err, "failed to query installation")
		}
		if installation == nil {
			return nil
		}

		err = printJSON(installation)
		if err != nil {
			return err
		}

		return nil
	},
}

var installationListCmd = &cobra.Command{
	Use:   "list",
	Short: "List created installations.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		owner, _ := command.Flags().GetString("owner")
		page, _ := command.Flags().GetInt("page")
		perPage, _ := command.Flags().GetInt("per-page")
		includeDeleted, _ := command.Flags().GetBool("include-deleted")
		installations, err := client.GetInstallations(&model.GetInstallationsRequest{
			OwnerID:        owner,
			Page:           page,
			PerPage:        perPage,
			IncludeDeleted: includeDeleted,
		})
		if err != nil {
			return errors.Wrap(err, "failed to query installations")
		}

		err = printJSON(installations)
		if err != nil {
			return err
		}

		return nil
	},
}
