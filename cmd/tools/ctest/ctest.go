// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/mattermost/mattermost-cloud/model"
	cloud "github.com/mattermost/mattermost-cloud/model"
	"github.com/spf13/cobra"
)

const (
	defaultLocalServerAPI = "http://localhost:8075"
	// DefaultPort is default listening port for incoming webhooks.
	DefaultPort = "8085"
	// ListenPortEnv is the env var name for overriding the default listen port.
	ListenPortEnv = "CTEST_PORT"
)

func init() {
	databaseCmd.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
	databaseCmd.PersistentFlags().String("webhook-url", "http://localhost:8085", "The listener URL of this tool which can be reached from the provisioner. (hint: use ngrok when not testing local provisioners)")
	databaseCmd.PersistentFlags().String("installation-domain", "dev.cloud.mattermost.com", "The provisioning server whose API will be queried.")
}

var databaseCmd = &cobra.Command{
	Use:   "database",
	Short: "Run the database test suite.",
	RunE: func(command *cobra.Command, args []string) error {
		port := DefaultPort
		if len(os.Getenv(ListenPortEnv)) != 0 {
			port = os.Getenv(ListenPortEnv)
		}
		serverAddress, _ := command.Flags().GetString("server")
		webhookURL, _ := command.Flags().GetString("webhook-url")
		installationDomain, _ := command.Flags().GetString("installation-domain")

		log.Printf("Starting cloud webhook listener on port %s", port)

		c := make(chan *model.WebhookPayload)
		go databaseTests(serverAddress, webhookURL, installationDomain, c)

		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			webhookHandler(w, r, c)
		})
		log.Fatal(http.ListenAndServe(":"+port, nil))

		return nil
	},
}

func webhookHandler(w http.ResponseWriter, r *http.Request, c chan *model.WebhookPayload) {
	webhook, err := cloud.WebhookPayloadFromReader(r.Body)
	if err != nil {
		log.Printf("Error: failed to parse webhook: %s", err)
		return
	}
	if len(webhook.ID) == 0 {
		return
	}

	wType := "UNKN"
	switch webhook.Type {
	case cloud.TypeCluster:
		wType = "CLSR"
	case cloud.TypeInstallation:
		wType = "INST"
	case cloud.TypeClusterInstallation:
		wType = "CLIN"
	}

	c <- webhook

	log.Printf("[ %s | %s ] %s -> %s", wType, webhook.ID[0:4], webhook.OldState, webhook.NewState)

	w.WriteHeader(http.StatusOK)
}

func databaseTests(serverAddress, webhookURL, installationDomain string, c chan *model.WebhookPayload) {
	databaseTypes := []string{
		model.InstallationDatabaseMultiTenantRDSPostgres,
		model.InstallationDatabaseMultiTenantRDSMySQL,
		model.InstallationDatabaseSingleTenantRDSPostgres,
		model.InstallationDatabaseSingleTenantRDSMySQL,
		model.InstallationDatabaseMysqlOperator,
	}

	client := model.NewClient(serverAddress)

	testWebhook, err := client.CreateWebhook(&model.CreateWebhookRequest{
		OwnerID: "ctest-filestore",
		URL:     webhookURL,
	})
	if err != nil {
		log.Printf("ERROR: unable to create test webhook: %s", err)
		return
	}
	log.Println("Test webhook created")

	deleteWebhook := func() {
		err := client.DeleteWebhook(testWebhook.ID)
		if err != nil {
			log.Printf("ERROR: unable to delete test webhook: %s", err)
			return
		}
		log.Println("Test webhook deleted")
	}
	defer deleteWebhook()

	testResults := []string{}
	printResults := func() {
		printSeparator()
		log.Println("DATABASE TEST RESULTS")
		printSeparator()
		for _, result := range testResults {
			log.Println(result)
		}
		printSeparator()
	}
	defer printResults()

	for _, databaseType := range databaseTypes {
		printSeparator()
		log.Printf("Running database test %s", databaseType)
		printSeparator()

		testStart := time.Now()

		installation, err := client.CreateInstallation(&model.CreateInstallationRequest{
			OwnerID:   "ctest-database-tests",
			DNS:       fmt.Sprintf("ctest-%s.%s", databaseType, installationDomain),
			Version:   "5.28.0",
			Affinity:  model.InstallationAffinityMultiTenant,
			Database:  databaseType,
			Filestore: model.InstallationFilestoreMultiTenantAwsS3,
		})
		if err != nil {
			log.Printf("ERROR: %s", err)
			testResults = append(testResults, fmt.Sprintf("FAIL: %s", databaseType))
			return
		}

		out, _ := json.Marshal(installation)
		log.Printf("Installation: %s", string(out))

		log.Printf("Waiting for installation %s to go stable", installation.ID)
		for {
			payload := <-c
			if payload.ID == installation.ID && payload.NewState == model.InstallationStateStable {
				log.Printf("Installation %s is now stable; tearing down installation", installation.ID)
				break
			}
		}

		err = client.DeleteInstallation(installation.ID)
		if err != nil {
			log.Printf("ERROR: %s", err)
			testResults = append(testResults, fmt.Sprintf("FAIL: %s", databaseType))
			return
		}

		log.Printf("Waiting for installation %s to be deleted", installation.ID)
		for {
			payload := <-c
			if payload.ID == installation.ID && payload.NewState == model.InstallationStateDeleted {
				log.Printf("Installation %s is now deleted", installation.ID)
				break
			}
		}

		now := time.Now()
		testMinutes := fmt.Sprintf("%.2f", now.Sub(testStart).Minutes())

		log.Printf("Database test %s completed in %s minutes", databaseType, testMinutes)
		testResults = append(testResults, fmt.Sprintf("PASS: %s (%s minutes)", databaseType, testMinutes))
	}

	log.Print("Tests Completed")
}

func printSeparator() {
	log.Print("====================================================")
}
