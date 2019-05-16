package main

import (
	"github.com/mattermost/mattermost-cloud/internal/api"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	installationCmd.PersistentFlags().String("server", "http://localhost:8075", "The provisioning server whose API will be queried.")

	installationCreateCmd.Flags().String("owner", "", "An opaque identifier describing the owner of the installation.")
	installationCreateCmd.Flags().String("version", "stable", "The Mattermost version to install.")
	installationCreateCmd.Flags().String("dns", "", "The URL at which the Mattermost server will be available.")
	installationCreateCmd.Flags().String("affinity", "isolated", "How other installations may be co-located in the same cluster.")
	installationCreateCmd.MarkFlagRequired("owner")
	installationCreateCmd.MarkFlagRequired("dns")

	installationUpgradeCmd.Flags().String("installation", "", "The id of the installation to be upgraded.")
	installationUpgradeCmd.Flags().String("version", "stable", "The Mattermost version to target.")
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
		client := api.NewClient(serverAddress)

		ownerID, _ := command.Flags().GetString("owner")
		version, _ := command.Flags().GetString("version")
		dns, _ := command.Flags().GetString("dns")
		affinity, _ := command.Flags().GetString("affinity")

		installation, err := client.CreateInstallation(&api.CreateInstallationRequest{
			OwnerID:  ownerID,
			Version:  version,
			DNS:      dns,
			Affinity: affinity,
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
		client := api.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		version, _ := command.Flags().GetString("version")

		err := client.UpgradeInstallation(installationID, version)
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
		client := api.NewClient(serverAddress)

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
		client := api.NewClient(serverAddress)

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
		client := api.NewClient(serverAddress)

		owner, _ := command.Flags().GetString("owner")
		page, _ := command.Flags().GetInt("page")
		perPage, _ := command.Flags().GetInt("per-page")
		includeDeleted, _ := command.Flags().GetBool("include-deleted")
		installations, err := client.GetInstallations(&api.GetInstallationsRequest{
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
