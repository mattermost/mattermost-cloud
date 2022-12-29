package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	stdlibHttp "net/http"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/model"
)

type ClusterTestSuite struct {
	http   *httpServer
	client *model.Client

	webhookEvents   chan *model.WebhookPayload
	webhookHandlers map[string]chan *model.WebhookPayload

	cancel context.CancelFunc
}

func (s *ClusterTestSuite) Client() *model.Client {
	return s.client
}

func (s *ClusterTestSuite) handleWebhookEvents() {
	for {
		payload := <-s.WebhookEvents()
		key := strings.Join([]string{string(payload.Type), payload.ID}, "-")
		if _, exists := s.webhookHandlers[key]; !exists {
			// Event without a handler
			continue
		}

		s.webhookHandlers[key] <- payload
	}
}

func (s *ClusterTestSuite) WebhookEvents() chan *model.WebhookPayload {
	return s.webhookEvents
}

func (s *ClusterTestSuite) RegisterWebhook() error {
	_, err := s.client.CreateWebhook(&model.CreateWebhookRequest{
		OwnerID: testIdentifier,
		URL:     webhookServerURL,
	})

	return err
}

func (s *ClusterTestSuite) UnregisterWebhook() error {
	webhoooks, err := s.client.GetWebhooks(&model.GetWebhooksRequest{
		OwnerID: testIdentifier,
	})
	if err != nil {
		return err
	}

	for _, webhook := range webhoooks {
		log.Println("Removing webhook " + webhook.ID)
		if err := s.client.DeleteWebhook(webhook.ID); err != nil {
			fmt.Printf("Error creating request webhook %s: %s", webhook.ID, err.Error())
			continue
		}
	}

	return nil
}

func (s *ClusterTestSuite) WaitForEvent(resource model.ResourceType, resourceID, expectedEvent, failedEvent string, timeout time.Duration) error {
	started := time.Now()

	key := strings.Join([]string{string(resource), resourceID}, "-")

	if _, exists := s.webhookHandlers[key]; exists {
		return fmt.Errorf("Trying to register handler twice %s", key)
	}

	ch := make(chan *model.WebhookPayload)
	s.webhookHandlers[key] = ch

	defer func() {
		close(ch)
		delete(s.webhookHandlers, key)
	}()

	log.Printf("Waiting for %s %s to be %s or %s for %v", resource, resourceID, expectedEvent, failedEvent, timeout)

	ticker := time.NewTicker(timeout)
	defer ticker.Stop()

	for {
		select {
		case payload := <-ch:
			switch payload.NewState {
			case expectedEvent:
				delta := time.Since(started)
				log.Printf("Expected %s event received after %v", expectedEvent, delta)
				return nil
			case failedEvent:
				return fmt.Errorf("received failed event: %s", payload.NewState)
			}
		case <-ticker.C:
			return fmt.Errorf("timeout")
		}
	}
}

func (s *ClusterTestSuite) StartServer(ctx context.Context) error {
	ctx, cancel := context.WithCancel(ctx)
	s.cancel = cancel

	go func() {
		if err := s.http.Start(ctx); err != nil && !errors.Is(err, stdlibHttp.ErrServerClosed) {
			log.Fatal("error starting server")
		}
	}()

	return nil
}

func (s *ClusterTestSuite) StopServer() {
	s.cancel()

	shuwdownContext, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.http.Stop(shuwdownContext); err != nil && !errors.Is(err, stdlibHttp.ErrServerClosed) {
		log.Println("error shutting down http server")
	}
}

func NewClusterTestSuite() *ClusterTestSuite {
	webhookPayloadChan := make(chan *model.WebhookPayload)

	server := &ClusterTestSuite{
		client:          model.NewClient(provisionerBaseURL),
		webhookEvents:   webhookPayloadChan,
		webhookHandlers: make(map[string]chan *model.WebhookPayload),
		http:            NewHttpServer(serverPort, webhookPayloadChan),
	}

	go server.handleWebhookEvents()

	return server
}
