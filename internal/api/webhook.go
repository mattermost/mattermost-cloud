// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/model"
)

// initWebhook registers webhook endpoints on the given router.
func initWebhook(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	webhooksRouter := apiRouter.PathPrefix("/webhooks").Subrouter()
	webhooksRouter.Handle("", addContext(handleGetWebhooks)).Methods("GET")
	webhooksRouter.Handle("", addContext(handleCreateWebhook)).Methods("POST")

	webhookRouter := apiRouter.PathPrefix("/webhook/{webhook:[A-Za-z0-9]{26}}").Subrouter()
	webhookRouter.Handle("", addContext(handleGetWebhook)).Methods("GET")
	webhookRouter.Handle("", addContext(handleDeleteWebhook)).Methods("DELETE")
}

// handleCreateWebhook responds to POST /api/webhooks, creating a new webhook.
func handleCreateWebhook(c *Context, w http.ResponseWriter, r *http.Request) {
	createWebhookRequest, err := model.NewCreateWebhookRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	webhook := model.Webhook{
		OwnerID: createWebhookRequest.OwnerID,
		URL:     createWebhookRequest.URL,
	}

	err = c.Store.CreateWebhook(&webhook)
	if err != nil {
		c.Logger.WithError(err).Error("failed to create webhook")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, webhook)
}

// handleGetWebhook responds to GET /api/webhook/{webhook}, returning the webhook in question.
func handleGetWebhook(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookID := vars["webhook"]
	c.Logger = c.Logger.WithField("webhook", webhookID)

	webhook, err := c.Store.GetWebhook(webhookID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if webhook == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, webhook)
}

// handleGetWebhooks responds to GET /api/webhooks, returning the specified page of webhooks.
func handleGetWebhooks(c *Context, w http.ResponseWriter, r *http.Request) {
	var err error
	owner := r.URL.Query().Get("owner")

	paging, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filter := &model.WebhookFilter{
		OwnerID: owner,
		Paging:  paging,
	}

	webhooks, err := c.Store.GetWebhooks(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query webhooks")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if webhooks == nil {
		webhooks = []*model.Webhook{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, webhooks)
}

// handleDeleteWebhook responds to DELETE /api/webhook/{webhook}, deleting the webhook.
func handleDeleteWebhook(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	webhookID := vars["webhook"]
	c.Logger = c.Logger.WithField("webhook", webhookID)

	webhook, err := c.Store.GetWebhook(webhookID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query webhook")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if webhook == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if webhook.IsDeleted() {
		c.Logger.Warn("unable to delete webhook that is already deleted")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = c.Store.DeleteWebhook(webhookID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to mark webhook as deleted")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
