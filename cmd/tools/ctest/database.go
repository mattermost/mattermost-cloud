// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"fmt"
	"time"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/spf13/cobra"
)

var databaseCmd = &cobra.Command{
	Use:   "database",
	Short: "Run the database test suite.",
	RunE: func(command *cobra.Command, args []string) error {
		port, _ := command.Flags().GetString("webhook-listener-port")

		c := make(chan *model.WebhookPayload)

		shutdown := startWebhookListener(port, c)
		defer shutdown()

		results := runDatabaseTests(command, c)
		printResults(results)

		return nil
	},
}

func runDatabaseTests(command *cobra.Command, c chan *model.WebhookPayload) []string {
	serverAddress, _ := command.Flags().GetString("server")
	webhookURL, _ := command.Flags().GetString("webhook-url")
	installationDomain, _ := command.Flags().GetString("installation-domain")
	version, _ := command.Flags().GetString("version")
	license, _ := command.Flags().GetString("license")

	databaseTypes := []string{
		model.InstallationDatabaseMultiTenantRDSPostgresPGBouncer,
		model.InstallationDatabaseMultiTenantRDSPostgres,
		model.InstallationDatabaseMultiTenantRDSMySQL,
		model.InstallationDatabaseSingleTenantRDSPostgres,
		model.InstallationDatabaseSingleTenantRDSMySQL,
	}

	client := model.NewClient(serverAddress)
	testResults := []string{}

	testWebhook, err := client.CreateWebhook(&model.CreateWebhookRequest{
		OwnerID: "ctest-database-tests",
		URL:     webhookURL,
	})
	if err != nil {
		logger.WithError(err).Error("Failed to create test webhook")
		return testResults
	}
	logger.Info("Test webhook created")

	deleteWebhook := func() {
		err := client.DeleteWebhook(testWebhook.ID)
		if err != nil {
			logger.WithError(err).Error("Failed to cleanup test webhook")
			return
		}
		logger.Info("Test webhook deleted")
	}
	defer deleteWebhook()

	for _, databaseType := range databaseTypes {
		printSeparator()
		logger.Infof("Running database test %s", databaseType)
		printSeparator()

		request := &model.CreateInstallationRequest{
			OwnerID:   "ctest-database-tests",
			DNS:       fmt.Sprintf("ctest-%s.%s", databaseType, installationDomain),
			Version:   version,
			License:   license,
			Affinity:  model.InstallationAffinityMultiTenant,
			Database:  databaseType,
			Filestore: model.InstallationFilestoreMultiTenantAwsS3,
		}

		testStart := time.Now()

		err := runInstallationLifecycleTest(request, client, c)
		if err != nil {
			logger.WithError(err).Error("Installation test failed")
			testResults = append(testResults, fmt.Sprintf("FAIL: %s", databaseType))
			return testResults
		}

		now := time.Now()
		testMinutes := fmt.Sprintf("%.2f", now.Sub(testStart).Minutes())

		logger.Infof("Database test %s completed in %s minutes", databaseType, testMinutes)
		testResults = append(testResults, fmt.Sprintf("PASS: %s (%s minutes)", databaseType, testMinutes))
	}

	logger.Info("Tests Completed")

	return testResults
}
