// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	clusterInstallationCmd.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")

	clusterInstallationGetCmd.Flags().String("cluster-installation", "", "The id of the cluster installation to be fetched.")
	clusterInstallationGetCmd.MarkFlagRequired("cluster-installation")

	clusterInstallationListCmd.Flags().String("cluster", "", "The cluster by which to filter cluster installations.")
	clusterInstallationListCmd.Flags().String("installation", "", "The installation by which to filter cluster installations.")
	registerTableOutputFlags(clusterInstallationListCmd)
	registerPagingFlags(clusterInstallationListCmd)

	clusterInstallationConfigCmd.PersistentFlags().String("cluster-installation", "", "The id of the cluster installation.")
	clusterInstallationConfigCmd.MarkFlagRequired("cluster-installation")

	clusterInstallationConfigSetCmd.Flags().String("key", "", "The configuration key to update (e.g. ServiceSettings.SiteURL).")
	clusterInstallationConfigSetCmd.Flags().String("value", "", "The value to write to the config.")
	clusterInstallationConfigSetCmd.MarkFlagRequired("key")
	clusterInstallationConfigSetCmd.MarkFlagRequired("value")

	clusterInstallationMMCTL.Flags().String("cluster-installation", "", "The id of the cluster installation.")
	clusterInstallationMMCTL.Flags().String("command", "", "The mmctl subcommand to run.")
	clusterInstallationMMCTL.MarkFlagRequired("cluster-installation")
	clusterInstallationMMCTL.MarkFlagRequired("command")

	clusterInstallationMattermostCLICmd.Flags().String("cluster-installation", "", "The id of the cluster installation.")
	clusterInstallationMattermostCLICmd.Flags().String("command", "", "The Mattermost CLI subcommand to run.")
	clusterInstallationMattermostCLICmd.MarkFlagRequired("cluster-installation")
	clusterInstallationMattermostCLICmd.MarkFlagRequired("command")

	clusterInstallationsMigrationCmd.Flags().String("source-cluster", "", "The source cluster for the migration to migrate cluster installations from.")
	clusterInstallationsMigrationCmd.MarkFlagRequired("source-cluster")
	clusterInstallationsMigrationCmd.Flags().String("target-cluster", "", "The target cluster for the migration to migrate cluster installation to.")
	clusterInstallationsMigrationCmd.MarkFlagRequired("target-cluster")
	clusterInstallationsMigrationCmd.Flags().String("installation", "", "The specific installation ID to migrate from source cluster, default is ALL.")

	dnsMigrationCmd.Flags().String("source-cluster", "", "The source cluster for the migration to switch CNAME(s) from.")
	dnsMigrationCmd.MarkFlagRequired("source-cluster")
	dnsMigrationCmd.Flags().String("target-cluster", "", "The target cluster for the migration to switch CNAME to.")
	dnsMigrationCmd.MarkFlagRequired("target-cluster")
	dnsMigrationCmd.Flags().String("installation", "", "The specific installation ID to migrate from source cluster, default is ALL.")
	dnsMigrationCmd.Flags().Bool("lock-installation", true, "The installation's lock flag during DNS migration process.")

	deleteInActiveClusterInstallationCmd.Flags().String("cluster", "", "The cluster ID to delete stale cluster installations from.")
	deleteInActiveClusterInstallationCmd.MarkFlagRequired("cluster")
	deleteInActiveClusterInstallationCmd.Flags().String("cluster-installation", "", "The id of the cluster installation.")

	postMigrationSwitchClusterRolesCmd.Flags().String("switch-role", "", "Post migration step to switch the roles for primary & secondary clusters.")
	postMigrationSwitchClusterRolesCmd.Flags().String("source-cluster", "", "The source cluster to be mark as secondary cluster.")
	postMigrationSwitchClusterRolesCmd.MarkFlagRequired("source-cluster")
	postMigrationSwitchClusterRolesCmd.Flags().String("target-cluster", "", "The target cluster to be mark as primary cluster.")
	postMigrationSwitchClusterRolesCmd.MarkFlagRequired("target-cluster")

	clusterInstallationCmd.AddCommand(clusterInstallationGetCmd)
	clusterInstallationCmd.AddCommand(clusterInstallationListCmd)
	clusterInstallationCmd.AddCommand(clusterInstallationConfigCmd)
	clusterInstallationCmd.AddCommand(clusterInstallationMMCTL)
	clusterInstallationCmd.AddCommand(clusterInstallationMattermostCLICmd)

	clusterInstallationsMigrationCmd.AddCommand(dnsMigrationCmd)
	clusterInstallationsMigrationCmd.AddCommand(deleteInActiveClusterInstallationCmd)
	clusterInstallationsMigrationCmd.AddCommand(postMigrationSwitchClusterRolesCmd)
	clusterInstallationCmd.AddCommand(clusterInstallationsMigrationCmd)

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
		paging := parsePagingFlags(command)

		clusterInstallations, err := client.GetClusterInstallations(&model.GetClusterInstallationsRequest{
			ClusterID:      cluster,
			InstallationID: installation,
			Paging:         paging,
		})
		if err != nil {
			return errors.Wrap(err, "failed to query cluster installations")
		}

		if enabled, customCols := tableOutputEnabled(command); enabled {
			var keys []string
			var vals [][]string

			if len(customCols) > 0 {
				data := make([]interface{}, 0, len(clusterInstallations))
				for _, inst := range clusterInstallations {
					data = append(data, inst)
				}
				keys, vals, err = prepareTableData(customCols, data)
				if err != nil {
					return errors.Wrap(err, "failed to prepare table output")
				}
			} else {
				keys, vals = defaultClusterInstallationTableData(clusterInstallations)
			}

			printTable(keys, vals)
			return nil
		}

		err = printJSON(clusterInstallations)
		if err != nil {
			return err
		}

		return nil
	},
}

func defaultClusterInstallationTableData(cis []*model.ClusterInstallation) ([]string, [][]string) {
	keys := []string{"ID", "STATE", "INSTALLATION ID", "CLUSTER ID"}
	vals := make([][]string, 0, len(cis))

	for _, ci := range cis {
		vals = append(vals, []string{ci.ID, ci.State, ci.InstallationID, ci.ClusterID})
	}
	return keys, vals
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

var clusterInstallationMMCTL = &cobra.Command{
	Use:   "mmctl",
	Short: "Run a mmctl command on a cluster installation",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterInstallationID, _ := command.Flags().GetString("cluster-installation")
		subcommand, _ := command.Flags().GetString("command")

		output, err := client.ExecClusterInstallationCLI(clusterInstallationID, "mmctl", strings.Split(subcommand, " "))

		// Print any output and then check and handle errors.
		fmt.Println(string(output))
		if err != nil {
			return errors.Wrap(err, "failed to run mattermost CLI command")
		}

		return nil
	},
}

var clusterInstallationMattermostCLICmd = &cobra.Command{
	Use:   "mattermost-cli",
	Short: "Run a mattermost CLI command on a cluster installation",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterInstallationID, _ := command.Flags().GetString("cluster-installation")
		subcommand, _ := command.Flags().GetString("command")

		output, err := client.RunMattermostCLICommandOnClusterInstallation(clusterInstallationID, strings.Split(subcommand, " "))

		// Print any output and then check and handle errors.
		fmt.Println(string(output))
		if err != nil {
			return errors.Wrap(err, "failed to run mattermost CLI command")
		}

		return nil
	},
}

// Command to migrate cluster installation(s)
var clusterInstallationsMigrationCmd = &cobra.Command{
	Use:   "migration",
	Short: "Migrate installation(s) to the target cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		sourceCluster, _ := command.Flags().GetString("source-cluster")
		targetcluster, _ := command.Flags().GetString("target-cluster")
		installation, _ := command.Flags().GetString("installation")

		response, err := client.MigrateClusterInstallation(
			&model.MigrateClusterInstallationRequest{
				SourceClusterID:  sourceCluster,
				TargetClusterID:  targetcluster,
				InstallationID:   installation,
				DNSSwitch:        false,
				LockInstallation: false})

		if err != nil {
			return errors.Wrap(err, "failed to migrate cluster installation(s)")
		}
		// Print any output and then check and handle errors.
		err = printJSON(response)
		if err != nil {
			return errors.Wrap(err, "failed to print cluster installation's migration response")
		}
		return nil
	},
}

// Command to migrate DNS record(s)
var dnsMigrationCmd = &cobra.Command{
	Use:   "dns migration",
	Short: "Switch over the DNS CNAME record(s) to the target cluster's Load Balancer.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		sourceCluster, _ := command.Flags().GetString("source-cluster")
		targetcluster, _ := command.Flags().GetString("target-cluster")
		installation, _ := command.Flags().GetString("installation")
		lockInstallation, _ := command.Flags().GetBool("lock-installation")

		response, err := client.MigrateDNS(
			&model.MigrateClusterInstallationRequest{
				SourceClusterID:  sourceCluster,
				TargetClusterID:  targetcluster,
				InstallationID:   installation,
				LockInstallation: lockInstallation})
		if err != nil {
			return errors.Wrap(err, "failed to perform DNS switch")
		}
		// Print any output and then check and handle errors.
		err = printJSON(response)
		if err != nil {
			return errors.Wrap(err, "failed to print DNS switch response")
		}
		return nil
	},
}

// Command to migrate DNS record(s)
var deleteInActiveClusterInstallationCmd = &cobra.Command{
	Use:   "delete stale cluster installation(s)",
	Short: "Delete stale cluster installation(s) after migration.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		cluster, _ := command.Flags().GetString("cluster")
		clusterInstallationID, _ := command.Flags().GetString("cluster-installation")
		if len(clusterInstallationID) != 0 {
			deletedCI, err := client.DeleteInActiveClusterInstallationByID(clusterInstallationID)
			if err != nil {
				return errors.Wrap(err, "failed to delete inactive cluster installations")
			}
			// Print any output and then check and handle errors.
			err = printJSON(deletedCI)
			if err != nil {
				return errors.Wrap(err, "failed to print deleting inactive cluster installation response")
			}
			return nil
		}

		if len(cluster) != 0 {
			response, err := client.DeleteInActiveClusterInstallationsByCluster(cluster)
			if err != nil {
				return errors.Wrap(err, "failed to delete inactive cluster installations")
			}
			// Print any output and then check and handle errors.
			err = printJSON(response)
			if err != nil {
				return errors.Wrap(err, "failed to print deleting inactive cluster installation response")
			}
			return nil
		}
		return nil
	},
}

// Command to switch cluster roles, mainly mark secondary cluster as primary after migration
var postMigrationSwitchClusterRolesCmd = &cobra.Command{
	Use:   "switch-cluster role",
	Short: "Mark the target/secondary cluster as primary cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		sourceCluster, _ := command.Flags().GetString("source-cluster")
		targetcluster, _ := command.Flags().GetString("target-cluster")
		installation, _ := command.Flags().GetString("installation")
		lockInstallation, _ := command.Flags().GetBool("lock-installation")

		mcir, err := client.SwitchClusterRoles(
			&model.MigrateClusterInstallationRequest{
				SourceClusterID:  sourceCluster,
				TargetClusterID:  targetcluster,
				InstallationID:   installation,
				LockInstallation: lockInstallation})
		if err != nil {
			return errors.Wrap(err, "failed to switch cluster roles")
		}

		// Print any output and then check and handle errors.
		err = printJSON(mcir)
		if err != nil {
			return errors.Wrap(err, "failed to print switch cluster roles response")
		}
		return nil
	},
}
