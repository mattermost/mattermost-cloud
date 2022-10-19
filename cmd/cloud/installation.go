// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"fmt"
	"os"
	"strings"

	sdkAWS "github.com/aws/aws-sdk-go/aws"
	toolsAWS "github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const hiddenLicense = "hidden (--hide-license=true)"

func init() {
	installationCmd.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
	installationCmd.PersistentFlags().Bool("dry-run", false, "When set to true, only print the API request without sending it.")

	installationCreateCmd.Flags().String("name", "", "Unique human-readable installation name. It should be the same as first segment of domain name.")
	installationCreateCmd.Flags().String("owner", "", "An opaque identifier describing the owner of the installation.")
	installationCreateCmd.Flags().String("group", "", "The id of the group to join")
	installationCreateCmd.Flags().String("version", "stable", "The Mattermost version to install.")
	installationCreateCmd.Flags().String("image", "mattermost/mattermost-enterprise-edition", "The Mattermost container image to use.")
	installationCreateCmd.Flags().StringSlice("dns", []string{}, "URLs at which the Mattermost server will be available.")

	installationCreateCmd.Flags().String("size", model.InstallationDefaultSize, "The size of the installation. Accepts 100users, 1000users, 5000users, 10000users, 25000users, miniSingleton, or miniHA. Defaults to 100users.")
	installationCreateCmd.Flags().String("affinity", model.InstallationAffinityIsolated, "How other installations may be co-located in the same cluster.")
	installationCreateCmd.Flags().String("license", "", "The Mattermost License to use in the server.")
	installationCreateCmd.Flags().String("database", model.InstallationDatabaseMysqlOperator, "The Mattermost server database type. Accepts mysql-operator, aws-rds, aws-rds-postgres, aws-multitenant-rds, or aws-multitenant-rds-postgres")
	installationCreateCmd.Flags().String("filestore", model.InstallationFilestoreMinioOperator, "The Mattermost server filestore type. Accepts minio-operator, aws-s3, bifrost, or aws-multitenant-s3")
	installationCreateCmd.Flags().StringArray("mattermost-env", []string{}, "Env vars to add to the Mattermost App. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	installationCreateCmd.Flags().StringArray("priority-env", []string{}, "Env vars to add to the Mattermost App that take priority over group config. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	installationCreateCmd.Flags().StringArray("annotation", []string{}, "Additional annotations for the installation. Accepts multiple values, for example: '... --annotation abc --annotation def'")
	installationCreateCmd.Flags().StringArray("group-selection-annotation", []string{}, "Annotations for automatic group selection. Accepts multiple values, for example: '... --group-selection-annotation abc --group-selection-annotation def'")
	installationCreateCmd.Flags().String("rds-primary-instance", "", "The machine instance type used for primary replica of database cluster. Works only with single tenant RDS databases.")
	installationCreateCmd.Flags().String("rds-replica-instance", "", "The machine instance type used for reader replicas of database cluster. Works only with single tenant RDS databases.")
	installationCreateCmd.Flags().Int("rds-replicas-count", 0, "The number of reader replicas of database cluster. Min: 0, Max: 15. Works only with single tenant RDS databases.")
	installationCreateCmd.Flags().String("external-database-secret-name", "", "The AWS secret name where the external database DSN is stored. Works only with external databases.")
	installationCreateCmd.MarkFlagRequired("owner")
	installationCreateCmd.MarkFlagRequired("dns")

	installationUpdateCmd.Flags().String("installation", "", "The id of the installation to be updated.")
	installationUpdateCmd.Flags().String("owner", "", "The new owner value of this installation.")
	installationUpdateCmd.Flags().String("image", "mattermost/mattermost-enterprise-edition", "The Mattermost container image to use.")
	installationUpdateCmd.Flags().String("version", "stable", "The Mattermost version to target.")
	installationUpdateCmd.Flags().String("size", model.InstallationDefaultSize, "The size of the installation. Accepts 100users, 1000users, 5000users, 10000users, 25000users, miniSingleton, or miniHA. Defaults to 100users.")
	installationUpdateCmd.Flags().String("license", "", "The Mattermost License to use in the server.")
	installationUpdateCmd.Flags().StringArray("mattermost-env", []string{}, "Env vars to add to the Mattermost App. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	installationUpdateCmd.Flags().StringArray("priority-env", []string{}, "Env vars to add to the Mattermost App that take priority over group config. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	installationUpdateCmd.Flags().Bool("mattermost-env-clear", false, "Clears all env var data.")
	installationUpdateCmd.Flags().Bool("priority-env-clear", false, "Clears all priority env var data.")
	installationUpdateCmd.MarkFlagRequired("installation")

	installationGetCmd.Flags().String("installation", "", "The id of the installation to be fetched.")
	installationGetCmd.Flags().Bool("include-group-config", true, "Whether to include group configuration in the installation or not.")
	installationGetCmd.Flags().Bool("include-group-config-overrides", true, "Whether to include a group configuration override summary in the installation or not.")
	installationGetCmd.Flags().Bool("hide-license", true, "Whether to hide the license value in the output or not.")
	installationGetCmd.Flags().Bool("hide-env", true, "Whether to hide env vars in the output or not.")
	installationGetCmd.MarkFlagRequired("installation")

	installationListCmd.Flags().String("owner", "", "The owner ID to filter installations by.")
	installationListCmd.Flags().String("group", "", "The group ID to filter installations.")
	installationListCmd.Flags().String("state", "", "The state to filter installations by.")
	installationListCmd.Flags().String("dns", "", "The dns name to filter installations by.")
	installationListCmd.Flags().Bool("include-group-config", true, "Whether to include group configuration in the installations or not.")
	installationListCmd.Flags().Bool("include-group-config-overrides", true, "Whether to include a group configuration override summary in the installations or not.")
	installationListCmd.Flags().Bool("hide-license", true, "Whether to hide the license value in the output or not.")
	installationListCmd.Flags().Bool("hide-env", true, "Whether to hide env vars in the output or not.")
	registerTableOutputFlags(installationListCmd)
	registerPagingFlags(installationListCmd)

	installationHibernateCmd.Flags().String("installation", "", "The id of the installation to put into hibernation.")
	installationHibernateCmd.MarkFlagRequired("installation")

	installationWakeupCmd.Flags().String("installation", "", "The id of the installation to wake up from hibernation.")
	installationWakeupCmd.Flags().String("owner", "", "The new owner value of this installation.")
	installationWakeupCmd.Flags().String("image", "mattermost/mattermost-enterprise-edition", "The Mattermost container image to use.")
	installationWakeupCmd.Flags().String("version", "stable", "The Mattermost version to target.")
	installationWakeupCmd.Flags().String("size", model.InstallationDefaultSize, "The size of the installation. Accepts 100users, 1000users, 5000users, 10000users, 25000users, miniSingleton, or miniHA. Defaults to 100users.")
	installationWakeupCmd.Flags().String("license", "", "The Mattermost License to use in the server.")
	installationWakeupCmd.Flags().StringArray("mattermost-env", []string{}, "Env vars to add to the Mattermost App. Accepts format: KEY_NAME=VALUE. Use the flag multiple times to set multiple env vars.")
	installationWakeupCmd.Flags().Bool("mattermost-env-clear", false, "Clears all env var data.")
	installationWakeupCmd.MarkFlagRequired("installation")

	installationDeleteCmd.Flags().String("installation", "", "The id of the installation to be deleted.")
	installationDeleteCmd.MarkFlagRequired("installation")

	installationCancelDeletionCmd.Flags().String("installation", "", "The id of the installation to cancel pending deletion for.")
	installationCancelDeletionCmd.MarkFlagRequired("installation")

	installationRecoveryCmd.Flags().String("installation", "", "The id of the installation to be recovered.")
	installationRecoveryCmd.Flags().String("installation-database", "", "The original multitenant database id of the installation to be recovered.")
	installationRecoveryCmd.Flags().String("database", "sqlite://cloud.db", "The database backing the provisioning server.")
	installationRecoveryCmd.MarkFlagRequired("installation")
	installationRecoveryCmd.MarkFlagRequired("installation-database")

	installationDeploymentReportCmd.Flags().String("installation", "", "The id of the installation to report on.")
	installationDeploymentReportCmd.MarkFlagRequired("installation")

	installationCmd.AddCommand(installationCreateCmd)
	installationCmd.AddCommand(installationUpdateCmd)
	installationCmd.AddCommand(installationDeleteCmd)
	installationCmd.AddCommand(installationCancelDeletionCmd)
	installationCmd.AddCommand(installationHibernateCmd)
	installationCmd.AddCommand(installationWakeupCmd)
	installationCmd.AddCommand(installationGetCmd)
	installationCmd.AddCommand(installationListCmd)
	installationCmd.AddCommand(installationShowStateReport)
	installationCmd.AddCommand(installationAnnotationCmd)
	installationCmd.AddCommand(installationsGetStatuses)
	installationCmd.AddCommand(installationRecoveryCmd)
	installationCmd.AddCommand(backupCmd)
	installationCmd.AddCommand(installationOperationCmd)
	installationCmd.AddCommand(installationDeploymentReportCmd)
	installationCmd.AddCommand(installationDNSCmd)
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

		name, _ := command.Flags().GetString("name")
		ownerID, _ := command.Flags().GetString("owner")
		groupID, _ := command.Flags().GetString("group")
		version, _ := command.Flags().GetString("version")
		image, _ := command.Flags().GetString("image")
		size, _ := command.Flags().GetString("size")
		dns, _ := command.Flags().GetStringSlice("dns")
		affinity, _ := command.Flags().GetString("affinity")
		license, _ := command.Flags().GetString("license")
		database, _ := command.Flags().GetString("database")
		filestore, _ := command.Flags().GetString("filestore")
		mattermostEnv, _ := command.Flags().GetStringArray("mattermost-env")
		priorityEnv, _ := command.Flags().GetStringArray("priority-env")
		annotations, _ := command.Flags().GetStringArray("annotation")
		groupSelectionAnnotations, _ := command.Flags().GetStringArray("group-selection-annotation")

		envVarMap, err := parseEnvVarInput(mattermostEnv, false)
		if err != nil {
			return err
		}
		priorityEnvVarMap, err := parseEnvVarInput(priorityEnv, false)
		if err != nil {
			return err
		}

		request := &model.CreateInstallationRequest{
			Name:                      name,
			OwnerID:                   ownerID,
			GroupID:                   groupID,
			Version:                   version,
			Image:                     image,
			Size:                      size,
			License:                   license,
			Affinity:                  affinity,
			Database:                  database,
			Filestore:                 filestore,
			MattermostEnv:             envVarMap,
			PriorityEnv:               priorityEnvVarMap,
			Annotations:               annotations,
			GroupSelectionAnnotations: groupSelectionAnnotations,
		}
		// For CLI to be backward compatible, if only one DNS is passed we use
		// the old field.
		if len(dns) == 1 {
			request.DNS = dns[0]
		} else {
			request.DNSNames = dns
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

		if database == model.InstallationDatabaseExternal {
			externalDatabaseSecretName, _ := command.Flags().GetString("external-database-secret-name")

			request.ExternalDatabaseConfig = model.ExternalDatabaseRequest{
				SecretName: externalDatabaseSecretName,
			}
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
		priorityEnv, _ := command.Flags().GetStringArray("priority-env")
		priorityEnvClear, _ := command.Flags().GetBool("priority-env-clear")

		envVarMap, err := parseEnvVarInput(mattermostEnv, mattermostEnvClear)
		if err != nil {
			return err
		}
		priorityEnvVarMap, err := parseEnvVarInput(priorityEnv, priorityEnvClear)
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
			PriorityEnv:   priorityEnvVarMap,
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

var installationCancelDeletionCmd = &cobra.Command{
	Use:   "cancel-deletion",
	Short: "Cancels the pending deletion of an installation.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")

		err := client.CancelInstallationDeletion(installationID)
		if err != nil {
			return errors.Wrap(err, "failed to cancel installation deletion")
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

		installation, err := client.WakeupInstallation(installationID, request)
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
		hideEnv, _ := command.Flags().GetBool("hide-env")

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
		if hideLicense {
			hideMattermostLicense(installation.Installation)
		}
		if hideEnv {
			hideMattermostEnv(installation.Installation)
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
		hideEnv, _ := command.Flags().GetBool("hide-env")
		paging := parsePagingFlags(command)

		installations, err := client.GetInstallations(&model.GetInstallationsRequest{
			OwnerID:                     owner,
			GroupID:                     group,
			State:                       state,
			DNS:                         dns,
			IncludeGroupConfig:          includeGroupConfig,
			IncludeGroupConfigOverrides: includeGroupConfigOverrides,
			Paging:                      paging,
		})
		if err != nil {
			return errors.Wrap(err, "failed to query installations")
		}

		if hideLicense {
			for _, installation := range installations {
				hideMattermostLicense(installation.Installation)
			}
		}

		if hideEnv {
			for _, installation := range installations {
				hideMattermostEnv(installation.Installation)
			}
		}

		if enabled, customCols := tableOutputEnabled(command); enabled {
			var keys []string
			var vals [][]string

			if len(customCols) > 0 {
				data := make([]interface{}, 0, len(installations))
				for _, inst := range installations {
					data = append(data, inst)
				}
				keys, vals, err = prepareTableData(customCols, data)
				if err != nil {
					return errors.Wrap(err, "failed to prepare table output")
				}
			} else {
				keys, vals = defaultInstallationTableData(installations)
			}

			printTable(keys, vals)
			return nil
		}

		err = printJSON(installations)
		if err != nil {
			return err
		}

		return nil
	},
}

func defaultInstallationTableData(installations []*model.InstallationDTO) ([]string, [][]string) {
	keys := []string{"ID", "STATE", "VERSION", "DATABASE", "FILESTORE", "DNS"}
	vals := make([][]string, 0, len(installations))
	for _, installation := range installations {
		vals = append(vals, []string{installation.ID, installation.State, installation.Version, installation.Database, installation.Filestore, dnsNames(installation.DNSRecords)})
	}
	return keys, vals
}

func dnsNames(dnsRecords []*model.InstallationDNS) string {
	names := model.DNSNamesFromRecords(dnsRecords)
	return strings.Join(names, ", ")
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

// WARNING: EXPERIMENTAL
// This command runs as a client with direct store integration.
var installationRecoveryCmd = &cobra.Command{
	Use:   "recover",
	Short: "recover the basic resources of a deleted installation by recreating it.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		installationID, _ := command.Flags().GetString("installation")
		databaseID, _ := command.Flags().GetString("installation-database")

		logger := logger.WithFields(logrus.Fields{
			"instance":              instanceID,
			"installation":          installationID,
			"installation-database": databaseID,
		})

		logger.Info("Starting installation recovery from deleted state")

		sqlStore, err := sqlStore(command)
		if err != nil {
			return errors.Wrap(err, "failed to create datastore")
		}

		installation, err := sqlStore.GetInstallation(installationID, true, false)
		if err != nil {
			return errors.Wrap(err, "failed to get installation")
		}
		if installation == nil {
			return errors.New("installation does not exist")
		}
		if installation.State != model.InstallationStateDeleted {
			return errors.New("installation recovery can only be performed on deleted installations")
		}

		// Name/DNS could have been claimed by a new installation, so we need to check
		// that as well.
		installations, err := sqlStore.GetInstallations(&model.InstallationFilter{
			Name:   installation.Name,
			Paging: model.AllPagesNotDeleted(),
		}, false, false)
		if err != nil {
			return errors.Wrap(err, "failed to get installations filtered by DNS")
		}
		if len(installations) != 0 {
			return errors.Errorf("the requested installation's DNS is now in use by %d installations", len(installations))
		}

		// Be extra cautious until we can test other configs.
		if installation.Database != model.InstallationDatabaseMultiTenantRDSPostgres {
			return errors.Errorf("installation database type %s is not supported", installation.Database)
		}
		if installation.Filestore != model.InstallationFilestoreBifrost && installation.Filestore != model.InstallationFilestoreMultiTenantAwsS3 {
			return errors.Errorf("installation filestore type %s is not supported", installation.Filestore)
		}

		clusterInstallations, err := sqlStore.GetClusterInstallations(&model.ClusterInstallationFilter{
			InstallationID: installation.ID,
			Paging:         model.AllPagesWithDeleted(),
		})
		if err != nil {
			return errors.Wrap(err, "failed to get cluster installations")
		}
		if len(clusterInstallations) != 1 {
			return errors.Errorf("expected to find 1 cluster installation, but found %d", len(clusterInstallations))
		}
		clusterInstallation := clusterInstallations[0]

		db, err := sqlStore.GetMultitenantDatabase(databaseID)
		if err != nil {
			return errors.Wrap(err, "failed to find database")
		}
		if db == nil {
			return errors.New("failed to find multitenant database")
		}

		logger.Info("Restoring AWS resources")

		awsRegion := os.Getenv("AWS_REGION")
		if awsRegion == "" {
			awsRegion = toolsAWS.DefaultAWSRegion
		}
		awsConfig := &sdkAWS.Config{
			Region:     sdkAWS.String(awsRegion),
			MaxRetries: sdkAWS.Int(toolsAWS.DefaultAWSClientRetries),
		}
		awsClient, err := toolsAWS.NewAWSClientWithConfig(awsConfig, logger)
		if err != nil {
			return errors.Wrap(err, "failed to build AWS client")
		}

		err = awsClient.SecretsManagerRestoreSecret(toolsAWS.RDSMultitenantSecretName(installation.ID), logger)
		if err != nil {
			return errors.Wrap(err, "failed to recover AWS database secret")
		}

		logger.Info("Updating multitenant database")

		locked, err := sqlStore.LockMultitenantDatabase(db.ID, instanceID)
		if err != nil {
			return errors.Wrap(err, "failed to lock multitenant database")
		}
		if !locked {
			return errors.New("failed to acquire lock on multitenant database")
		}
		defer func() {
			unlocked, err := sqlStore.UnlockMultitenantDatabase(db.ID, instanceID, false)
			if err != nil {
				logger.WithError(err).Error("Failed to unlock multitenant database")
				return
			}
			if !unlocked {
				logger.Error("Failed to release lock for multitenant database")
			}
		}()

		// Refresh the database object in case updates were made.
		db, err = sqlStore.GetMultitenantDatabase(databaseID)
		if err != nil {
			return errors.Wrap(err, "failed to get database")
		}

		// Handle follow-up attempts from a partial recovery.
		// NOTE: this ignores DB weighting.
		if !db.Installations.Contains(installationID) {
			db.Installations.Add(installationID)
			err = sqlStore.UpdateMultitenantDatabase(db)
			if err != nil {
				return errors.Wrap(err, "failed to add installation ID to multitenant database")
			}
		}

		logger.Infof("Setting cluster installation %s to creation-requested", clusterInstallation.ID)

		// We shouldn't need to lock this as it is a single update and nothing
		// should have a lock since it is deleted.
		if clusterInstallation.State == model.ClusterInstallationStateDeleted {
			clusterInstallation.State = model.ClusterInstallationStateCreationRequested
			err = sqlStore.RecoverClusterInstallation(clusterInstallation)
			if err != nil {
				return errors.Wrap(err, "failed to set cluster installation to creation-requested state")
			}
		}

		logger.Info("Setting installation to creation-requested")

		installation.State = model.InstallationStateCreationRequested
		err = sqlStore.RecoverInstallation(installation)
		if err != nil {
			return errors.Wrap(err, "failed to set installation to creation-requested state")
		}

		logger.Info("Installation recovery request completed")

		return nil
	},
}

var installationDeploymentReportCmd = &cobra.Command{
	Use:   "report",
	Short: "Get a report of deployment details for a given installation",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")

		installation, err := client.GetInstallation(installationID, &model.GetInstallationRequest{
			IncludeGroupConfig:          true,
			IncludeGroupConfigOverrides: true,
		})
		if err != nil {
			return errors.Wrap(err, "failed to query installation")
		}
		if installation == nil {
			return nil
		}

		output := fmt.Sprintf("Installation: %s\n", installation.ID)
		output += fmt.Sprintf(" ├ Created: %s\n", installation.CreationDateString())
		output += fmt.Sprintf(" ├ State: %s\n", installation.State)
		if installation.State == model.InstallationStateDeleted {
			output += fmt.Sprintf(" │ └ Deleted: %s\n", installation.DeletionDateString())
		}
		output += fmt.Sprintf(" ├ DNS: %s\n", installation.DNS)
		output += fmt.Sprintf(" ├ Version: %s:%s\n", installation.Image, installation.Version)
		output += fmt.Sprintf(" ├ Size: %s\n", installation.Size)
		output += fmt.Sprintf(" ├ Affinity: %s\n", installation.Affinity)
		output += fmt.Sprintf(" ├ Environment Variables: %d\n", len(installation.MattermostEnv))
		if len(installation.PriorityEnv) != 0 {
			var envKeys []string
			for key := range installation.PriorityEnv {
				envKeys = append(envKeys, key)
			}
			output += fmt.Sprintf(" ├ Priority Environment Variables: %d\n", len(installation.PriorityEnv))
			output += fmt.Sprintf(" │ └ Env Keys Set: %s\n", strings.Join(envKeys, ", "))
		}
		output += fmt.Sprintf(" ├ Database Type: %s\n", installation.Database)
		if model.IsMultiTenantRDS(installation.Database) {
			databases, err := client.GetMultitenantDatabases(&model.GetMultitenantDatabasesRequest{
				Paging: model.AllPagesWithDeleted(),
			})
			if err != nil {
				return errors.Wrap(err, "failed to query installation database")
			}
			for _, database := range databases {
				if database.Installations.Contains(installation.ID) {
					output += fmt.Sprintf(" │ ├ Database: %s\n", database.ID)
					output += fmt.Sprintf(" │ ├ State: %s\n", database.State)
					output += fmt.Sprintf(" │ ├ VPC: %s\n", database.VpcID)
					output += fmt.Sprintf(" │ ├ Database Writer Endpoint: %s\n", database.WriterEndpoint)
					output += fmt.Sprintf(" │ └ Installations: %d\n", len(database.Installations))
				}
			}
			if installation.Database == model.InstallationDatabaseMultiTenantRDSPostgresPGBouncer {
				schemas, err := client.GetDatabaseSchemas(&model.GetDatabaseSchemaRequest{
					InstallationID: installationID,
					Paging:         model.AllPagesWithDeleted(),
				})
				if err != nil {
					return errors.Wrap(err, "failed to query installation database schema")
				}
				for _, schema := range schemas {
					logicalDatabase, err := client.GetLogicalDatabase(schema.LogicalDatabaseID)
					if err != nil {
						return errors.Wrap(err, "failed to query installation logical database")
					}
					schemasInLogicalDatabase, err := client.GetDatabaseSchemas(&model.GetDatabaseSchemaRequest{
						LogicalDatabaseID: logicalDatabase.ID,
						Paging:            model.AllPagesNotDeleted(),
					})
					if err != nil {
						return errors.Wrap(err, "failed to query schemas in logical database")
					}
					output += fmt.Sprintf(" │   ├ Logical Database: %s\n", logicalDatabase.ID)
					output += fmt.Sprintf(" │   ├ Name: %s\n", logicalDatabase.Name)
					output += fmt.Sprintf(" │   └ Schemas: %d\n", len(schemasInLogicalDatabase))
					output += fmt.Sprintf(" │     ├ Database Schema: %s\n", schema.ID)
					output += fmt.Sprintf(" │     └ Name: %s\n", schema.Name)
				}
			}
		} else {
			switch installation.Database {
			case model.InstallationDatabaseSingleTenantRDSMySQL,
				model.InstallationDatabaseSingleTenantRDSPostgres:
				output += fmt.Sprintf(" │ ├ Primary Instance Type: %s\n", installation.SingleTenantDatabaseConfig.PrimaryInstanceType)
				output += fmt.Sprintf(" │ ├ Replica Instance Type: %s\n", installation.SingleTenantDatabaseConfig.ReplicaInstanceType)
				output += fmt.Sprintf(" │ └ Replica Count: %d\n", installation.SingleTenantDatabaseConfig.ReplicasCount)
			case model.InstallationDatabaseExternal:
				output += fmt.Sprintf(" │ └ Database Secret Name: %s\n", installation.ExternalDatabaseConfig.SecretName)
			}
		}
		output += fmt.Sprintf(" ├ Filestore Type: %s\n", installation.Filestore)

		if installation.GroupID != nil && len(*installation.GroupID) != 0 {
			group, err := client.GetGroup(*installation.GroupID)
			if err != nil {
				return errors.Wrap(err, "failed to query installation group")
			}
			output += fmt.Sprintf(" ├ Group: %s\n", group.ID)
			output += fmt.Sprintf(" │ ├ Name: %s\n", group.Name)
			output += fmt.Sprintf(" │ └ Description: %s\n", group.Description)
		}

		clusterInstallations, err := client.GetClusterInstallations(&model.GetClusterInstallationsRequest{
			InstallationID: installation.ID,
			Paging:         model.AllPagesWithDeleted(),
		})
		if err != nil {
			return errors.Wrap(err, "failed to query cluster installations")
		}
		for _, clusterInstallation := range clusterInstallations {
			output += fmt.Sprintf(" └ Cluster Installation: %s\n", clusterInstallation.ID)

			cluster, err := client.GetCluster(clusterInstallation.ClusterID)
			if err != nil {
				return errors.Wrap(err, "failed to query cluster")
			}
			output += fmt.Sprintf("   ├ State: %s\n", clusterInstallation.State)
			output += fmt.Sprintf("   └ Cluster: %s\n", cluster.ID)
			output += fmt.Sprintf("     ├ State: %s\n", cluster.State)
			output += fmt.Sprintf("     ├ VPC: %s\n", cluster.ProvisionerMetadataKops.VPC)
			output += fmt.Sprintf("     ├ Nodes: Masters %d, Workers %d\n", cluster.ProvisionerMetadataKops.MasterCount, cluster.ProvisionerMetadataKops.NodeMaxCount)
			output += fmt.Sprintf("     └ Version: %s\n", cluster.ProvisionerMetadataKops.Version)
		}

		fmt.Println(output)

		return nil
	},
}

func hideMattermostLicense(installation *model.Installation) {
	if len(installation.License) != 0 {
		installation.License = hiddenLicense
	}
}

func hideMattermostEnv(installation *model.Installation) {
	if installation.MattermostEnv != nil {
		installation.MattermostEnv = model.EnvVarMap{
			fmt.Sprintf("hidden (%d env vars) (--hide-env=true)", len(installation.MattermostEnv)): model.EnvVar{},
		}
	}
}
