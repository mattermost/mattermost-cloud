package main

import (
	"strings"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	installationCmd.PersistentFlags().String("server", "http://localhost:8075", "The provisioning server whose API will be queried.")

	installationCreateCmd.Flags().String("owner", "", "An opaque identifier describing the owner of the installation.")
	installationCreateCmd.Flags().String("group", "", "The id of the group to join")
	installationCreateCmd.Flags().String("version", "stable", "The Mattermost version to install.")
	installationCreateCmd.Flags().String("image", "mattermost/mattermost-enterprise-edition", "The Mattermost container image to use.")
	installationCreateCmd.Flags().String("dns", "", "The URL at which the Mattermost server will be available.")
	installationCreateCmd.Flags().String("size", model.InstallationDefaultSize, "The size of the installation. Accepts 100users, 1000users, 5000users, 10000users, 25000users, miniSingleton, or miniHA. Defaults to 100users.")
	installationCreateCmd.Flags().String("affinity", model.InstallationAffinityIsolated, "How other installations may be co-located in the same cluster.")
	installationCreateCmd.Flags().String("license", "", "The Mattermost License to use in the server.")
	installationCreateCmd.Flags().String("database", model.InstallationDatabaseMysqlOperator, "The Mattermost server database type. Accepts mysql-operator or aws-rds")
	installationCreateCmd.Flags().String("filestore", model.InstallationFilestoreMinioOperator, "The Mattermost server filestore type. Accepts minio-operator or aws-s3")
	installationCreateCmd.Flags().StringArray("mattermost-env", []string{}, "Env vars to add to the Mattermost App. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	installationCreateCmd.MarkFlagRequired("owner")
	installationCreateCmd.MarkFlagRequired("dns")

	installationUpdateCmd.Flags().String("installation", "", "The id of the installation to be updated.")
	installationUpdateCmd.Flags().String("version", "stable", "The Mattermost version to target.")
	installationUpdateCmd.Flags().String("image", "mattermost/mattermost-enterprise-edition", "The Mattermost container image to use.")
	installationUpdateCmd.Flags().String("license", "", "The Mattermost License to use in the server.")
	installationUpdateCmd.Flags().StringArray("mattermost-env", []string{}, "Env vars to add to the Mattermost App. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	installationUpdateCmd.MarkFlagRequired("installation")

	installationDeleteCmd.Flags().String("installation", "", "The id of the installation to be deleted.")
	installationDeleteCmd.MarkFlagRequired("installation")

	installationGetCmd.Flags().String("installation", "", "The id of the installation to be fetched.")
	installationGetCmd.Flags().Bool("include-group-config", true, "Whether to include group configuration in the installation or not.")
	installationGetCmd.Flags().Bool("include-group-config-overrides", true, "Whether to include a group configuration override summary in the installation or not.")
	installationGetCmd.MarkFlagRequired("installation")

	installationListCmd.Flags().String("owner", "", "The owner by which to filter installations.")
	installationListCmd.Flags().String("group", "", "The group ID by which to filter installations.")
	installationListCmd.Flags().Bool("include-group-config", true, "Whether to include group configuration in the installations or not.")
	installationListCmd.Flags().Bool("include-group-config-overrides", true, "Whether to include a group configuration override summary in the installations or not.")
	installationListCmd.Flags().Int("page", 0, "The page of installations to fetch, starting at 0.")
	installationListCmd.Flags().Int("per-page", 100, "The number of installations to fetch per page.")
	installationListCmd.Flags().Bool("include-deleted", false, "Whether to include deleted installations.")

	installationCmd.AddCommand(installationCreateCmd)
	installationCmd.AddCommand(installationUpdateCmd)
	installationCmd.AddCommand(installationDeleteCmd)
	installationCmd.AddCommand(installationGetCmd)
	installationCmd.AddCommand(installationListCmd)
	installationCmd.AddCommand(installationShowStateReport)
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
		groupID, _ := command.Flags().GetString("group")
		version, _ := command.Flags().GetString("version")
		image, _ := command.Flags().GetString("image")
		size, _ := command.Flags().GetString("size")
		dns, _ := command.Flags().GetString("dns")
		affinity, _ := command.Flags().GetString("affinity")
		license, _ := command.Flags().GetString("license")
		database, _ := command.Flags().GetString("database")
		filestore, _ := command.Flags().GetString("filestore")
		mattermostEnv, _ := command.Flags().GetStringArray("mattermost-env")

		envVarMap, err := parseEnvVarInput(mattermostEnv)
		if err != nil {
			return err
		}

		installation, err := client.CreateInstallation(&model.CreateInstallationRequest{
			OwnerID:       ownerID,
			GroupID:       groupID,
			Version:       version,
			Image:         image,
			Size:          size,
			DNS:           dns,
			License:       license,
			Affinity:      affinity,
			Database:      database,
			Filestore:     filestore,
			MattermostEnv: envVarMap,
		})
		if err != nil {
			return errors.Wrap(err, "failed to create installation")
		}

		return printJSON(installation)
	},
}

var installationUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update an installation's configuration",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		mattermostEnv, _ := command.Flags().GetStringArray("mattermost-env")

		envVarMap, err := parseEnvVarInput(mattermostEnv)
		if err != nil {
			return err
		}

		installation, err := client.UpdateInstallation(
			installationID,
			&model.PatchInstallationRequest{
				Version:       getStringFlagPointer(command, "version"),
				Image:         getStringFlagPointer(command, "image"),
				License:       getStringFlagPointer(command, "license"),
				MattermostEnv: envVarMap,
			},
		)
		if err != nil {
			return errors.Wrap(err, "failed to update installation")
		}

		return printJSON(installation)
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
		includeGroupConfig, _ := command.Flags().GetBool("include-group-config")
		includeGroupConfigOverrides, _ := command.Flags().GetBool("include-group-config-overrides")

		installation, err := client.GetInstallation(installationID, &model.GetInstallationRequest{
			IncludeGroupConfig:          includeGroupConfig,
			IncludeGroupConfigOverrides: includeGroupConfigOverrides,
		})
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
		group, _ := command.Flags().GetString("group")
		includeGroupConfig, _ := command.Flags().GetBool("include-group-config")
		includeGroupConfigOverrides, _ := command.Flags().GetBool("include-group-config-overrides")
		page, _ := command.Flags().GetInt("page")
		perPage, _ := command.Flags().GetInt("per-page")
		includeDeleted, _ := command.Flags().GetBool("include-deleted")
		installations, err := client.GetInstallations(&model.GetInstallationsRequest{
			OwnerID:                     owner,
			GroupID:                     group,
			IncludeGroupConfig:          includeGroupConfig,
			IncludeGroupConfigOverrides: includeGroupConfigOverrides,
			Page:                        page,
			PerPage:                     perPage,
			IncludeDeleted:              includeDeleted,
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

// TODO:
// Instead of showing the state data from the model of the CLI binary, add a new
// API endpoint to return the server's state model.
var installationShowStateReport = &cobra.Command{
	Use:   "state-report",
	Short: "Shows information regarding changing installation state.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		err := printJSON(model.GetInstallationRequestStateReport())
		if err != nil {
			return err
		}

		return nil
	},
}

func parseEnvVarInput(rawInput []string) (model.EnvVarMap, error) {
	if len(rawInput) == 0 {
		return nil, nil
	}

	envVarMap := make(model.EnvVarMap)

	for _, env := range rawInput {
		// Split the input once by "=" to allow for multiple "="s to be in the
		// value. Expect there to still be one key and value.
		kv := strings.SplitN(env, "=", 2)
		if len(kv) != 2 || len(kv[0]) == 0 || len(kv[1]) == 0 {
			return nil, errors.Errorf("%s is not in a valid env format; expecting KEY_NAME=VALUE", env)
		}

		if _, ok := envVarMap[kv[0]]; ok {
			return nil, errors.Errorf("env var %s was defined more than once", kv[0])
		}

		envVarMap[kv[0]] = model.EnvVar{Value: kv[1]}
	}

	return envVarMap, nil
}
