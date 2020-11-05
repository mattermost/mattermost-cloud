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

var filestoreCmd = &cobra.Command{
	Use:   "filestore",
	Short: "Run the filestore test suite.",
	RunE: func(command *cobra.Command, args []string) error {
		port, _ := command.Flags().GetString("webhook-listener-port")

		logger.Infof("Starting cloud webhook listener on port %s", port)

		c := make(chan *model.WebhookPayload)
		go filestoreTests(command, c)

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			webhookHandler(w, r, c)
		})
		logger.Fatal(http.ListenAndServe(":"+port, nil))

		return nil
	},
}

func filestoreTests(command *cobra.Command, c chan *model.WebhookPayload) {
	serverAddress, _ := command.Flags().GetString("server")
	webhookURL, _ := command.Flags().GetString("webhook-url")
	installationDomain, _ := command.Flags().GetString("installation-domain")
	version, _ := command.Flags().GetString("version")
	license, _ := command.Flags().GetString("license")

	filestoreTypes := []string{
		model.InstallationFilestoreBifrost,
		model.InstallationFilestoreMultiTenantAwsS3,
		model.InstallationFilestoreAwsS3,
		model.InstallationFilestoreMinioOperator,
	}

	client := model.NewClient(serverAddress)

	testWebhook, err := client.CreateWebhook(&model.CreateWebhookRequest{
		OwnerID: "ctest-filestore-tests",
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
		logger.Info("FILESTORE TEST RESULTS")
		printSeparator()
		for _, result := range testResults {
			logger.Info(result)
		}
		printSeparator()
	}
	defer printResults()

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
			return
		}

		now := time.Now()
		testMinutes := fmt.Sprintf("%.2f", now.Sub(testStart).Minutes())

		logger.Infof("Filestore test %s completed in %s minutes", filestoreType, testMinutes)
		testResults = append(testResults, fmt.Sprintf("PASS: %s (%s minutes)", filestoreType, testMinutes))
	}

	logger.Info("Tests Completed")
}
