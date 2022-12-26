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

var filestoreCmd = &cobra.Command{
	Use:   "filestore",
	Short: "Run the filestore test suite.",
	RunE: func(cmd *cobra.Command, args []string) error {
		port, _ := cmd.Flags().GetString("webhook-listener-port")

		c := make(chan *model.WebhookPayload)

		shutdown := startWebhookListener(port, c)
		defer shutdown()

		results := runFilestoreTests(cmd, c)
		printResults(results)

		return nil
	},
}

func runFilestoreTests(cmd *cobra.Command, c chan *model.WebhookPayload) []string {
	serverAddress, _ := cmd.Flags().GetString("server")
	webhookURL, _ := cmd.Flags().GetString("webhook-url")
	installationDomain, _ := cmd.Flags().GetString("installation-domain")
	version, _ := cmd.Flags().GetString("version")
	license, _ := cmd.Flags().GetString("license")

	filestoreTypes := []string{
		model.InstallationFilestoreBifrost,
		model.InstallationFilestoreMultiTenantAwsS3,
		model.InstallationFilestoreAwsS3,
		model.InstallationFilestoreMinioOperator,
	}

	client := model.NewClient(serverAddress)
	testResults := []string{}

	testWebhook, err := client.CreateWebhook(&model.CreateWebhookRequest{
		OwnerID: "ctest-filestore-tests",
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

	for _, filestoreType := range filestoreTypes {
		printSeparator()
		logger.Infof("Running filestore test %s", filestoreType)
		printSeparator()

		request := &model.CreateInstallationRequest{
			OwnerID:   "ctest-filestore-tests",
			DNS:       fmt.Sprintf("ctest-%s.%s", filestoreType, installationDomain),
			Version:   version,
			License:   license,
			Affinity:  model.InstallationAffinityMultiTenant,
			Database:  model.InstallationDatabaseMultiTenantRDSPostgres,
			Filestore: filestoreType,
		}

		testStart := time.Now()

		err := runInstallationLifecycleTest(request, client, c)
		if err != nil {
			logger.WithError(err).Error("Installation test failed")
			testResults = append(testResults, fmt.Sprintf("FAIL: %s", filestoreType))
			return testResults
		}

		now := time.Now()
		testMinutes := fmt.Sprintf("%.2f", now.Sub(testStart).Minutes())

		logger.Infof("Filestore test %s completed in %s minutes", filestoreType, testMinutes)
		testResults = append(testResults, fmt.Sprintf("PASS: %s (%s minutes)", filestoreType, testMinutes))
	}

	logger.Info("Tests Completed")

	return testResults
}
