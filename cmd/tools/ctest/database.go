// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/spf13/cobra"
)

var databaseCmd = &cobra.Command{
	Use:   "database",
	Short: "Run the database test suite.",
	RunE: func(command *cobra.Command, args []string) error {
		port, _ := command.Flags().GetString("webhook-listener-port")

		logger.Infof("Starting cloud webhook listener on port %s", port)

		c := make(chan *model.WebhookPayload)
		go databaseTests(command, c)

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			webhookHandler(w, r, c)
		})
		logger.Fatal(http.ListenAndServe(":"+port, nil))

		return nil
	},
}

func databaseTests(command *cobra.Command, c chan *model.WebhookPayload) {
	serverAddress, _ := command.Flags().GetString("server")
	webhookURL, _ := command.Flags().GetString("webhook-url")
	installationDomain, _ := command.Flags().GetString("installation-domain")
	version, _ := command.Flags().GetString("version")

	databaseTypes := []string{
		model.InstallationDatabaseMultiTenantRDSPostgres,
		model.InstallationDatabaseMultiTenantRDSMySQL,
		model.InstallationDatabaseSingleTenantRDSPostgres,
		model.InstallationDatabaseSingleTenantRDSMySQL,
		model.InstallationDatabaseMysqlOperator,
	}

	client := model.NewClient(serverAddress)

	testWebhook, err := client.CreateWebhook(&model.CreateWebhookRequest{
		OwnerID: "ctest-database-tests",
		URL:     webhookURL,
	})
	if err != nil {
		logger.WithError(err).Error("Failed to create test webhook")
		return
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

	testResults := []string{}
	printResults := func() {
		printSeparator()
		logger.Info("DATABASE TEST RESULTS")
		printSeparator()
		for _, result := range testResults {
			logger.Info(result)
		}
		printSeparator()
	}
	defer printResults()

	for _, databaseType := range databaseTypes {
		printSeparator()
		logger.Infof("Running database test %s", databaseType)
		printSeparator()

		request := &model.CreateInstallationRequest{
			OwnerID:   "ctest-database-tests",
			DNS:       fmt.Sprintf("ctest-%s.%s", databaseType, installationDomain),
			Version:   version,
			Affinity:  model.InstallationAffinityMultiTenant,
			Database:  databaseType,
			Filestore: model.InstallationFilestoreMultiTenantAwsS3,
		}

		testStart := time.Now()

		err := runInstallationLifecycleTest(request, client, c)
		if err != nil {
			logger.WithError(err).Error("Installation test failed")
			testResults = append(testResults, fmt.Sprintf("FAIL: %s", databaseType))
			return
		}

		now := time.Now()
		testMinutes := fmt.Sprintf("%.2f", now.Sub(testStart).Minutes())

		logger.Infof("Database test %s completed in %s minutes", databaseType, testMinutes)
		testResults = append(testResults, fmt.Sprintf("PASS: %s (%s minutes)", databaseType, testMinutes))
	}

	logger.Info("Tests Completed")
}
