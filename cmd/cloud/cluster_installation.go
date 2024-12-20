// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newCmdClusterInstallation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "installation",
		Short: "Manipulate cluster installations managed by the provisioning server.",
	}

	cmd.AddCommand(newCmdClusterInstallationGet())
	cmd.AddCommand(newCmdClusterInstallationList())
	cmd.AddCommand(newCmdClusterInstallationConfig())
	cmd.AddCommand(newCmdClusterInstallationStatus())
	cmd.AddCommand(newCmdClusterInstallationMMCTL())
	cmd.AddCommand(newCmdClusterInstallationMattermostCLI())
	cmd.AddCommand(newCmdClusterInstallationPPROF())
	cmd.AddCommand(newCmdClusterInstallationMigration())

	return cmd
}

func newCmdClusterInstallationGet() *cobra.Command {
	var flags clusterInstallationGetFlags

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a particular cluster installation.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := createClient(command.Context(), flags.clusterFlags)

			clusterInstallation, err := client.GetClusterInstallation(flags.clusterInstallationID)
			if err != nil {
				return errors.Wrap(err, "failed to query cluster installation")
			}
			if clusterInstallation == nil {
				return nil
			}

			return printJSON(clusterInstallation)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func newCmdClusterInstallationList() *cobra.Command {
	var flags clusterInstallationListFlags

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List created cluster installations.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			return executeClusterInstallationListCmd(command.Context(), flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterInstallationListCmd(ctx context.Context, flags clusterInstallationListFlags) error {
	client := createClient(ctx, flags.clusterFlags)

	paging := getPaging(flags.pagingFlags)

	clusterInstallations, err := client.GetClusterInstallations(&model.GetClusterInstallationsRequest{
		ClusterID:      flags.cluster,
		InstallationID: flags.installation,
		Paging:         paging,
	})
	if err != nil {
		return errors.Wrap(err, "failed to query cluster installations")
	}

	if enabled, customCols := getTableOutputOption(flags.tableOptions); enabled {
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

	return printJSON(clusterInstallations)
}

func defaultClusterInstallationTableData(cis []*model.ClusterInstallation) ([]string, [][]string) {
	keys := []string{"ID", "STATE", "INSTALLATION ID", "CLUSTER ID"}
	vals := make([][]string, 0, len(cis))

	for _, ci := range cis {
		vals = append(vals, []string{ci.ID, ci.State, ci.InstallationID, ci.ClusterID})
	}
	return keys, vals
}

func newCmdClusterInstallationConfig() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manipulate a particular cluster installation's config.",
	}

	cmd.AddCommand(newCmdClusterInstallationConfigGet())
	cmd.AddCommand(newCmdClusterInstallationConfigSet())

	return cmd
}

func newCmdClusterInstallationConfigGet() *cobra.Command {
	var flags clusterInstallationConfigGetFlags

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a particular cluster installation's config.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := createClient(command.Context(), flags.clusterFlags)

			clusterInstallationConfig, err := client.GetClusterInstallationConfig(flags.clusterInstallationID)
			if err != nil {
				return errors.Wrap(err, "failed to query cluster installation config")
			}
			if clusterInstallationConfig == nil {
				return nil
			}

			return printJSON(clusterInstallationConfig)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func newCmdClusterInstallationConfigSet() *cobra.Command {
	var flags clusterInstallationConfigSetFlags

	cmd := &cobra.Command{
		Use:   "set",
		Short: "Set a particular cluster installation's config.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := createClient(command.Context(), flags.clusterFlags)

			config := make(map[string]interface{})
			keyParts := strings.Split(flags.key, ".")
			configRef := config
			for i, keyPart := range keyParts {
				if i < len(keyParts)-1 {
					value := make(map[string]interface{})
					configRef[keyPart] = value
					configRef = value
				} else {
					configRef[keyPart] = flags.val
				}
			}

			if err := client.SetClusterInstallationConfig(flags.clusterInstallationID, config); err != nil {
				return errors.Wrap(err, "failed to modify cluster installation config")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func newCmdClusterInstallationStatus() *cobra.Command {
	var flags clusterInstallationStatusFlags

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Get the status of a particular cluster installation.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := createClient(command.Context(), flags.clusterFlags)

			clusterInstallation, err := client.GetClusterInstallationStatus(flags.clusterInstallationID)
			if err != nil {
				return errors.Wrap(err, "failed to query cluster installation")
			}
			if clusterInstallation == nil {
				return nil
			}

			return printJSON(clusterInstallation)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func newCmdClusterInstallationMMCTL() *cobra.Command {
	var flags clusterInstallationMMCTLFlags

	cmd := &cobra.Command{
		Use:   "mmctl",
		Short: "Run a mmctl command on a cluster installation",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := createClient(command.Context(), flags.clusterFlags)

			output, err := client.ExecClusterInstallationCLI(flags.clusterInstallationID, "mmctl", strings.Split(flags.subcommand, " "))
			fmt.Println(string(output))
			if err != nil {
				return errors.Wrap(err, "failed to run mattermost CLI command")
			}

			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func newCmdClusterInstallationMattermostCLI() *cobra.Command {
	var flags clusterInstallationMattermostCLIFlags

	cmd := &cobra.Command{
		Use:   "mattermost-cli",
		Short: "Run a mattermost CLI command on a cluster installation",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := createClient(command.Context(), flags.clusterFlags)

			output, err := client.RunMattermostCLICommandOnClusterInstallation(flags.clusterInstallationID, strings.Split(flags.subcommand, " "))
			fmt.Println(string(output))
			if err != nil {
				return errors.Wrap(err, "failed to run mattermost CLI command")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func newCmdClusterInstallationPPROF() *cobra.Command {
	var flags clusterInstallationPPROFFlags

	cmd := &cobra.Command{
		Use:   "pprof",
		Short: "Gather pprof data from a cluster installation",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := createClient(command.Context(), flags.clusterFlags)

			output, err := client.ExecClusterInstallationPPROF(flags.clusterInstallationID)
			if err != nil {
				return errors.Wrap(err, "failed to run mattermost CLI command")
			}
			if output == nil {
				return errors.Wrap(err, "no debug data returned")
			}

			filename := fmt.Sprintf("%s.%s.prof.zip", flags.clusterInstallationID, time.Now().Format("2006-01-02.15-04-05.MST"))
			err = os.WriteFile(filename, output, 0644)
			if err != nil {
				return errors.Wrap(err, "failed to save debug zip")
			}

			fmt.Printf("Debug data saved to %s\n", filename)

			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func newCmdClusterInstallationMigration() *cobra.Command {
	var flags clusterInstallationMigrationFlags

	cmd := &cobra.Command{
		Use:   "migration",
		Short: "Migrate installation(s) to the target cluster.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			return executeClusterInstallationMigrationCmd(command.Context(), flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)
	cmd.AddCommand(newCmdClusterInstallationDNSMigration())
	cmd.AddCommand(newCmdDeleteInActiveClusterInstallation())
	cmd.AddCommand(newCmdClusterRolesPostMigrationSwitch())

	return cmd
}

func executeClusterInstallationMigrationCmd(ctx context.Context, flags clusterInstallationMigrationFlags) error {

	client := createClient(ctx, flags.clusterFlags)

	response, err := client.MigrateClusterInstallation(
		&model.MigrateClusterInstallationRequest{
			SourceClusterID:  flags.sourceCluster,
			TargetClusterID:  flags.targetCluster,
			InstallationID:   flags.installation,
			DNSSwitch:        false,
			LockInstallation: false})

	if err != nil {
		return errors.Wrap(err, "failed to migrate cluster installation(s)")
	}

	if err := printJSON(response); err != nil {
		return errors.Wrap(err, "failed to print cluster installation's migration response")
	}
	return nil
}

func newCmdClusterInstallationDNSMigration() *cobra.Command {
	var flags clusterInstallationDNSMigrationFlags

	cmd := &cobra.Command{
		Use:   "dns",
		Short: "Switch over the DNS CNAME record(s) to the target cluster's Load Balancer.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			return executeClusterInstallationDNSMigrationCmd(command.Context(), flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterInstallationDNSMigrationCmd(ctx context.Context, flags clusterInstallationDNSMigrationFlags) error {
	client := createClient(ctx, flags.clusterFlags)

	response, err := client.MigrateDNS(
		&model.MigrateClusterInstallationRequest{
			SourceClusterID:  flags.sourceCluster,
			TargetClusterID:  flags.targetCluster,
			InstallationID:   flags.installation,
			LockInstallation: flags.lockInstallation})
	if err != nil {
		return errors.Wrap(err, "failed to perform DNS switch")
	}

	if err := printJSON(response); err != nil {
		return errors.Wrap(err, "failed to print DNS switch response")
	}
	return nil
}

func newCmdDeleteInActiveClusterInstallation() *cobra.Command {
	var flags inActiveClusterInstallationDeleteFlags

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete stale cluster installation(s) after migration.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			return executeDeleteInActiveClusterInstallationCmd(command.Context(), flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeDeleteInActiveClusterInstallationCmd(ctx context.Context, flags inActiveClusterInstallationDeleteFlags) error {

	client := createClient(ctx, flags.clusterFlags)

	if len(flags.clusterInstallationID) != 0 {
		deletedCI, err := client.DeleteInActiveClusterInstallationByID(flags.clusterInstallationID)
		if err != nil {
			return errors.Wrap(err, "failed to delete inactive cluster installations")
		}

		if err := printJSON(deletedCI); err != nil {
			return errors.Wrap(err, "failed to print deleting inactive cluster installation response")
		}
		return nil
	}

	if len(flags.cluster) != 0 {
		response, err := client.DeleteInActiveClusterInstallationsByCluster(flags.cluster)
		if err != nil {
			return errors.Wrap(err, "failed to delete inactive cluster installations")
		}
		if err := printJSON(response); err != nil {
			return errors.Wrap(err, "failed to print deleting inactive cluster installation response")
		}
		return nil
	}
	return nil
}

func newCmdClusterRolesPostMigrationSwitch() *cobra.Command {
	var flags clusterRolesPostMigrationSwitchFlags

	cmd := &cobra.Command{
		Use:   "switch-cluster",
		Short: "Mark the target/secondary cluster as primary cluster.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			return executeClusterRolesPostMigrationSwitchCmd(command.Context(), flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterRolesPostMigrationSwitchCmd(ctx context.Context, flags clusterRolesPostMigrationSwitchFlags) error {
	client := createClient(ctx, flags.clusterFlags)

	response, err := client.SwitchClusterRoles(
		&model.MigrateClusterInstallationRequest{
			SourceClusterID: flags.sourceCluster,
			TargetClusterID: flags.targetCluster,
		})
	if err != nil {
		return errors.Wrap(err, "failed to switch cluster roles")
	}

	if err := printJSON(response); err != nil {
		return errors.Wrap(err, "failed to print switch cluster roles response")
	}
	return nil
}
