package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/mattermost/mattermost-cloud/model"
)

type httpServer struct {
	http *http.Server
}

func (s *httpServer) Start(_ context.Context) error {
	log.Printf("starting http server at %s", s.http.Addr)
	return s.http.ListenAndServe()
}

func (s *httpServer) Stop(ctx context.Context) error {
	log.Println("stoppping http server")
	return s.http.Shutdown(ctx)
}

func NewHttpServer(port int, webhookPayloadChan chan *model.WebhookPayload) *httpServer {
	mux := http.NewServeMux()

	mux.HandleFunc(webhookHandlerPath, func(w http.ResponseWriter, r *http.Request) {
		var payload *model.WebhookPayload
		body, _ := io.ReadAll(r.Body)
		if err := json.Unmarshal(body, &payload); err != nil {
			log.Fatal(err)
		}
		webhookPayloadChan <- payload
		w.WriteHeader(200)
	})

	return &httpServer{
		http: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
	}
}
