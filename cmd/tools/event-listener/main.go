// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

const (
	// DefaultPort is default listening port for incoming events.
	DefaultPort = "8099"
	// ListenPortEnv is the env var name for overriding the default listen port.
	ListenPortEnv = "EVENTS_PORT"
	// DefaultSubscriptionOwner is default owner ID assigned to subscription.
	DefaultSubscriptionOwner = "local-event-listener"
	// SubscriptionOwnerEnv is the env var name for overriding the default subscription owner.
	SubscriptionOwnerEnv = "SUB_OWNER"
	// CleanupSubEnvVar is env var name for overriding subscription cleanup.
	CleanupSubEnvVar = "CLEANUP_SUB"
)

func main() {
	port := DefaultPort
	if os.Getenv(ListenPortEnv) != "" {
		port = os.Getenv(ListenPortEnv)
	}
	subOwner := DefaultSubscriptionOwner
	if os.Getenv(SubscriptionOwnerEnv) != "" {
		subOwner = os.Getenv(SubscriptionOwnerEnv)
	}
	cleanup := true
	if os.Getenv(CleanupSubEnvVar) != "" {
		cleanupEnv := os.Getenv(CleanupSubEnvVar)
		c, err := strconv.ParseBool(cleanupEnv)
		if err != nil {
			log.WithError(err).Fatalf("Failed to parse CLEANUP_SUB env var to boolean, env value: %s", cleanupEnv)
			return
		}
		cleanup = c
	}

	log.Infof("Starting cloud webhook listener on port %s", port)
	if cleanup {
		log.Infof("Subscription cleanup will be attempted")
	}

	client := model.NewClient("http://localhost:8075")

	sub, err := ensureSubscription(client, port, subOwner)
	if err != nil {
		log.WithError(err).Error("Failed to ensure subscription")
		return
	}

	if cleanup {
		defer func() {
			log.Info("Attempting subscription cleanup.")
			err = client.DeleteSubscription(sub.ID)
			if err != nil {
				log.WithError(err).Error("Failed to cleanup subscription")
			}
		}()
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	http.HandleFunc("/", eventsHandler)
	server := &http.Server{
		Addr:    fmt.Sprintf(":%s", port),
		Handler: http.DefaultServeMux,
	}
	go func() {
		err = server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			log.WithError(err).Errorf("Error listening")
			close(sigChan)
			return
		}
	}()

	<-sigChan
	err = server.Close()
	if err != nil {
		log.WithError(err).Errorf("Failed to close server")
	}
}

func eventsHandler(w http.ResponseWriter, r *http.Request) {
	eventBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.Printf("Error: failed to read event: %s", err)
		return
	}
	formattedBody := &bytes.Buffer{}
	err = json.Indent(formattedBody, eventBody, "", "\t")
	if err != nil {
		log.WithError(err).Errorf("Failed to indent JSON")
		formattedBody = bytes.NewBuffer(eventBody)
	}

	fmt.Println("==================================")
	fmt.Println("Received Event:")
	fmt.Println(formattedBody.String())
	fmt.Println("==================================")

	w.WriteHeader(http.StatusOK)
}

func ensureSubscription(client *model.Client, port, owner string) (*model.Subscription, error) {
	listReq := model.ListSubscriptionsRequest{
		Paging: model.AllPagesNotDeleted(),
		Owner:  owner,
	}
	subs, err := client.ListSubscriptions(&listReq)
	if err != nil {
		return nil, fmt.Errorf("failed to list subscriptions: %w", err)
	}

	expectedURL := listenerURL(port)
	var sub *model.Subscription
	for _, s := range subs {
		if s.URL == expectedURL {
			sub = s
			log.WithField("subscription", sub.ID).Infof("Found existing subscription for Local Event Listener")
			break
		}
	}
	if sub == nil {
		sub, err = registerSubscription(client, port, owner)
		if err != nil {
			return nil, fmt.Errorf("failed to register subscription: %w", err)
		}
		log.WithField("subscription", sub.ID).Info("Registered new subscription")
	}
	return sub, nil
}

func registerSubscription(client *model.Client, port, owner string) (*model.Subscription, error) {
	createSubReq := model.CreateSubscriptionRequest{
		Name:             "Local Event Listener",
		URL:              listenerURL(port),
		OwnerID:          owner,
		EventType:        model.ResourceStateChangeEventType,
		FailureThreshold: 30 * time.Second, // Should guarantee one retry
	}

	return client.CreateSubscription(&createSubReq)
}

func listenerURL(port string) string {
	return fmt.Sprintf("http://localhost:%s", port)
}
