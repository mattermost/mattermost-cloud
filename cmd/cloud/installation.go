// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"os"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const hiddenLicense = "hidden (--hide-license=true)"

func init() {
	installationCmd.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
	installationCmd.PersistentFlags().Bool("dry-run", false, "When set to true, only print the API request without sending it.")

	installationCreateCmd.Flags().String("owner", "", "An opaque identifier describing the owner of the installation.")
	installationCreateCmd.Flags().String("group", "", "The id of the group to join")
	installationCreateCmd.Flags().String("version", "stable", "The Mattermost version to install.")
	installationCreateCmd.Flags().String("image", "mattermost/mattermost-enterprise-edition", "The Mattermost container image to use.")
	installationCreateCmd.Flags().String("dns", "", "The URL at which the Mattermost server will be available.")
	installationCreateCmd.Flags().String("size", model.InstallationDefaultSize, "The size of the installation. Accepts 100users, 1000users, 5000users, 10000users, 25000users, miniSingleton, or miniHA. Defaults to 100users.")
	installationCreateCmd.Flags().String("affinity", model.InstallationAffinityIsolated, "How other installations may be co-located in the same cluster.")
	installationCreateCmd.Flags().String("license", "", "The Mattermost License to use in the server.")
	installationCreateCmd.Flags().String("database", model.InstallationDatabaseMysqlOperator, "The Mattermost server database type. Accepts mysql-operator, aws-rds, aws-rds-postgres, or aws-multitenant-rds")
	installationCreateCmd.Flags().String("filestore", model.InstallationFilestoreMinioOperator, "The Mattermost server filestore type. Accepts minio-operator, aws-s3, bifrost, or aws-multitenant-s3")
	installationCreateCmd.Flags().StringArray("mattermost-env", []string{}, "Env vars to add to the Mattermost App. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	installationCreateCmd.Flags().StringArray("annotation", []string{}, "Additional annotations for the installation. Accepts multiple values, for example: '... --annotation abc --annotation def'")
	installationCreateCmd.Flags().String("rds-primary-instance", "", "The machine instance type used for primary replica of database cluster. Works only with single tenant RDS databases.")
	installationCreateCmd.Flags().String("rds-replica-instance", "", "The machine instance type used for reader replicas of database cluster. Works only with single tenant RDS databases.")
	installationCreateCmd.Flags().Int("rds-replicas-count", 0, "The number of reader replicas of database cluster. Min: 0, Max: 15. Works only with single tenant RDS databases.")
	installationCreateCmd.MarkFlagRequired("owner")
	installationCreateCmd.MarkFlagRequired("dns")

	installationUpdateCmd.Flags().String("installation", "", "The id of the installation to be updated.")
	installationUpdateCmd.Flags().String("owner", "", "The new owner value of this installation.")
	installationUpdateCmd.Flags().String("image", "mattermost/mattermost-enterprise-edition", "The Mattermost container image to use.")
	installationUpdateCmd.Flags().String("version", "stable", "The Mattermost version to target.")
	installationUpdateCmd.Flags().String("size", model.InstallationDefaultSize, "The size of the installation. Accepts 100users, 1000users, 5000users, 10000users, 25000users, miniSingleton, or miniHA. Defaults to 100users.")
	installationUpdateCmd.Flags().String("license", "", "The Mattermost License to use in the server.")
	installationUpdateCmd.Flags().StringArray("mattermost-env", []string{}, "Env vars to add to the Mattermost App. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	installationUpdateCmd.Flags().Bool("mattermost-env-clear", false, "Clears all env var data.")
	installationUpdateCmd.MarkFlagRequired("installation")

	installationGetCmd.Flags().String("installation", "", "The id of the installation to be fetched.")
	installationGetCmd.Flags().Bool("include-group-config", true, "Whether to include group configuration in the installation or not.")
	installationGetCmd.Flags().Bool("include-group-config-overrides", true, "Whether to include a group configuration override summary in the installation or not.")
	installationGetCmd.Flags().Bool("hide-license", true, "Whether to hide the license value in the output or not.")
	installationGetCmd.MarkFlagRequired("installation")

	installationListCmd.Flags().String("owner", "", "The owner ID to filter installations by.")
	installationListCmd.Flags().String("group", "", "The group ID to filter installations.")
	installationListCmd.Flags().String("state", "", "The state to filter installations by.")
	installationListCmd.Flags().String("dns", "", "The dns name to filter installations by.")
	installationListCmd.Flags().Bool("include-group-config", true, "Whether to include group configuration in the installations or not.")
	installationListCmd.Flags().Bool("include-group-config-overrides", true, "Whether to include a group configuration override summary in the installations or not.")
	installationListCmd.Flags().Bool("hide-license", true, "Whether to hide the license value in the output or not.")
	installationListCmd.Flags().Int("page", 0, "The page of installations to fetch, starting at 0.")
	installationListCmd.Flags().Int("per-page", 100, "The number of installations to fetch per page.")
	installationListCmd.Flags().Bool("include-deleted", false, "Whether to include deleted installations.")
	installationListCmd.Flags().Bool("table", false, "Whether to display the returned installation list in a table or not.")

	installationHibernateCmd.Flags().String("installation", "", "The id of the installation to put into hibernation.")
	installationHibernateCmd.MarkFlagRequired("installation")

	installationWakeupCmd.Flags().String("installation", "", "The id of the installation to wake up from hibernation.")
	installationWakeupCmd.MarkFlagRequired("installation")

	installationDeleteCmd.Flags().String("installation", "", "The id of the installation to be deleted.")
	installationDeleteCmd.MarkFlagRequired("installation")

	installationCmd.AddCommand(installationCreateCmd)
	installationCmd.AddCommand(installationUpdateCmd)
	installationCmd.AddCommand(installationDeleteCmd)
	installationCmd.AddCommand(installationHibernateCmd)
	installationCmd.AddCommand(installationWakeupCmd)
	installationCmd.AddCommand(installationGetCmd)
	installationCmd.AddCommand(installationListCmd)
	installationCmd.AddCommand(installationShowStateReport)
	installationCmd.AddCommand(installationAnnotationCmd)
	installationCmd.AddCommand(installationsGetStatuses)
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
		annotations, _ := command.Flags().GetStringArray("annotation")

		envVarMap, err := parseEnvVarInput(mattermostEnv, false)
		if err != nil {
			return err
		}

		request := &model.CreateInstallationRequest{
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
			Annotations:   annotations,
		}

		if model.IsSingleTenantRDS(database) {
			rdsPrimaryInstance, _ := command.Flags().GetString("rds-primary-instance")
			rdsReplicaInstance, _ := command.Flags().GetString("rds-replica-instance")
			rdsReplicasCount, _ := command.Flags().GetInt("rds-replicas-count")

			dbConfig := model.SingleTenantDatabaseRequest{
				PrimaryInstanceType: rdsPrimaryInstance,
				ReplicaInstanceType: rdsReplicaInstance,
				ReplicasCount:       rdsReplicasCount,
			}

			request.SingleTenantDatabaseConfig = dbConfig
		}

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			err = printJSON(request)
			if err != nil {
				return errors.Wrap(err, "failed to print API request")
			}

			return nil
		}

		installation, err := client.CreateInstallation(request)
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
		mattermostEnvClear, _ := command.Flags().GetBool("mattermost-env-clear")

		envVarMap, err := parseEnvVarInput(mattermostEnv, mattermostEnvClear)
		if err != nil {
			return err
		}

		request := &model.PatchInstallationRequest{
			OwnerID:       getStringFlagPointer(command, "owner"),
			Version:       getStringFlagPointer(command, "version"),
			Image:         getStringFlagPointer(command, "image"),
			Size:          getStringFlagPointer(command, "size"),
			License:       getStringFlagPointer(command, "license"),
			MattermostEnv: envVarMap,
		}

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			err = printJSON(request)
			if err != nil {
				return errors.Wrap(err, "failed to print API request")
			}

			return nil
		}

		installation, err := client.UpdateInstallation(installationID, request)
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

var installationHibernateCmd = &cobra.Command{
	Use:   "hibernate",
	Short: "Put an installation into hibernation.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")

		installation, err := client.HibernateInstallation(installationID)
		if err != nil {
			return errors.Wrap(err, "failed to put installation into hibernation")
		}

		err = printJSON(installation)
		if err != nil {
			return err
		}

		return nil
	},
}

var installationWakeupCmd = &cobra.Command{
	Use:   "wake-up",
	Short: "Wake an installation from hibernation.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")

		installation, err := client.WakeupInstallation(installationID)
		if err != nil {
			return errors.Wrap(err, "failed to wake up installation")
		}

		err = printJSON(installation)
		if err != nil {
			return err
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
		hideLicense, _ := command.Flags().GetBool("hide-license")

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
		if hideLicense && len(installation.License) != 0 {
			installation.License = hiddenLicense
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
		state, _ := command.Flags().GetString("state")
		dns, _ := command.Flags().GetString("dns")
		includeGroupConfig, _ := command.Flags().GetBool("include-group-config")
		includeGroupConfigOverrides, _ := command.Flags().GetBool("include-group-config-overrides")
		hideLicense, _ := command.Flags().GetBool("hide-license")
		page, _ := command.Flags().GetInt("page")
		perPage, _ := command.Flags().GetInt("per-page")
		includeDeleted, _ := command.Flags().GetBool("include-deleted")
		installations, err := client.GetInstallations(&model.GetInstallationsRequest{
			OwnerID:                     owner,
			GroupID:                     group,
			State:                       state,
			DNS:                         dns,
			IncludeGroupConfig:          includeGroupConfig,
			IncludeGroupConfigOverrides: includeGroupConfigOverrides,
			Page:                        page,
			PerPage:                     perPage,
			IncludeDeleted:              includeDeleted,
		})
		if err != nil {
			return errors.Wrap(err, "failed to query installations")
		}

		if hideLicense {
			for _, installation := range installations {
				if len(installation.License) != 0 {
					installation.License = hiddenLicense
				}
			}
		}

		outputToTable, _ := command.Flags().GetBool("table")
		if outputToTable {
			table := tablewriter.NewWriter(os.Stdout)
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.SetHeader([]string{"ID", "STATE", "VERSION", "DATABASE", "FILESTORE", "DNS"})

			for _, installation := range installations {
				table.Append([]string{installation.ID, installation.State, installation.Version, installation.Database, installation.Filestore, installation.DNS})
			}
			table.Render()

			return nil
		}

		err = printJSON(installations)
		if err != nil {
			return err
		}

		return nil
	},
}

var installationsGetStatuses = &cobra.Command{
	Use:   "status",
	Short: "Get status information for all installations.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationsStatus, err := client.GetInstallationsStatus()
		if err != nil {
			return errors.Wrap(err, "failed to query installation status")
		}
		if installationsStatus == nil {
			return nil
		}

		err = printJSON(installationsStatus)
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
