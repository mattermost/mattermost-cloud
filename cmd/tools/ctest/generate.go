// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	generateCmd.PersistentFlags().Int("count", 1, "Number of installations to generate.")
	generateCmd.PersistentFlags().String("dns-prefix", "ctest-", "Prefix text to prepend to all DNS names created.")
	generateCmd.PersistentFlags().String("group", "", "Specify the group to create installations in.")
	generateCmd.PersistentFlags().String("database", model.InstallationDatabaseMultiTenantRDSPostgres, "Specify the type of database with which to create installations.")
	generateCmd.PersistentFlags().String("filestore", model.InstallationFilestoreBifrost, "Specify the filestore type with which to create installations.")
	generateCmd.PersistentFlags().String("size", mmv1alpha1.CloudSize10String, "Specify the size of the created installations.")
}

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Generate multiple installations with a common configuration.",
	RunE: func(command *cobra.Command, args []string) error {
		return runGenerateCommand(command)
	},
}

func runGenerateCommand(command *cobra.Command) error {
	serverAddress, _ := command.Flags().GetString("server")
	dnsPrefix, _ := command.Flags().GetString("dns-prefix")
	installationDomain, _ := command.Flags().GetString("installation-domain")
	version, _ := command.Flags().GetString("version")
	group, _ := command.Flags().GetString("group")
	license, _ := command.Flags().GetString("license")
	database, _ := command.Flags().GetString("database")
	filestore, _ := command.Flags().GetString("filestore")
	size, _ := command.Flags().GetString("size")
	count, _ := command.Flags().GetInt("count")

	client := model.NewClient(serverAddress)

	printSeparator()
	logger.Infof("Generating %d installations", count)
	printSeparator()

	for i := 1; i <= count; i++ {
		installation, err := client.CreateInstallation(&model.CreateInstallationRequest{
			OwnerID:   "ctest-installation-generator",
			DNS:       fmt.Sprintf("%s%d.%s", dnsPrefix, i, installationDomain),
			GroupID:   group,
			Version:   version,
			License:   license,
			Database:  database,
			Filestore: filestore,
			Size:      size,
			Affinity:  model.InstallationAffinityMultiTenant,
		})
		if err != nil {
			return errors.Wrap(err, "failed to create installation")
		}
		logger.Debugf("%s - %s", installation.ID, installation.DNS)

		time.Sleep(100 * time.Millisecond)
	}

	logger.Info("Installation generation complete")

	return nil
}
