// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"log"
	"net/http"
	"os"

	cloud "github.com/mattermost/mattermost-cloud/model"
)

const (
	// DefaultPort is default listening port for incoming webhooks.
	DefaultPort = "8065"
	// ListenPortEnv is the env var name for overriding the default listen port.
	ListenPortEnv = "CWL_PORT"
)

func handler(w http.ResponseWriter, r *http.Request) {
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

	log.Printf("[ %s | %s ] %s -> %s", wType, webhook.ID[0:4], webhook.OldState, webhook.NewState)

	w.WriteHeader(http.StatusOK)
}

func main() {
	port := DefaultPort
	if len(os.Getenv(ListenPortEnv)) != 0 {
		port = os.Getenv(ListenPortEnv)
	}

	log.Printf("Starting cloud webhook listener on port %s", port)

	http.HandleFunc("/", handler)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
