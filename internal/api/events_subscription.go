// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/model"
)

// initSubscription registers subscription endpoints on the given router.
func initSubscription(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	subscriptionsRouter := apiRouter.PathPrefix("/subscriptions").Subrouter()

	subscriptionsRouter.Handle("", addContext(handleListSubscriptions)).Methods("GET")
	subscriptionsRouter.Handle("", addContext(handleRegisterSubscription)).Methods("POST")

	subscriptionRouter := apiRouter.PathPrefix("/subscription/{subscription:[A-Za-z0-9]{26}}").Subrouter()
	subscriptionRouter.Handle("", addContext(handleGetSubscription)).Methods("GET")
	subscriptionRouter.Handle("", addContext(handleDeleteSubscription)).Methods("DELETE")
}

// handleRegisterSubscription responds to POST /api/subscriptions, registering new subscription.
func handleRegisterSubscription(c *Context, w http.ResponseWriter, r *http.Request) {
	createSubReq, err := model.NewCreateSubscriptionRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sub, err := createSubReq.ToSubscription()
	if err != nil {
		c.Logger.WithError(err).Error("Request is invalid")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = c.Store.CreateSubscription(&sub)
	if err != nil {
		c.Logger.WithError(err).Error("failed to create subscription")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, sub)
}

// handleListSubscriptions responds to GET /api/subscriptions, returning the specified page of subscriptions.
func handleListSubscriptions(c *Context, w http.ResponseWriter, r *http.Request) {
	paging, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filter := &model.SubscriptionsFilter{
		Paging:    paging,
		Owner:     r.URL.Query().Get("owner"),
		EventType: model.EventType(r.URL.Query().Get("event_type")),
	}

	subscriptions, err := c.Store.GetSubscriptions(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query subscriptions")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if subscriptions == nil {
		subscriptions = []*model.Subscription{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, subscriptions)
}

// handleGetSubscription responds to GET /api/subscription/{subscription}, returning the subscription in question.
func handleGetSubscription(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subID := vars["subscription"]
	c.Logger = c.Logger.WithField("subscription", subID)

	subscription, err := c.Store.GetSubscription(subID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query subscription")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if subscription == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, subscription)
}

// handleDeleteSubscription responds to DELETE /api/subscription/{subscription}, deleting the subscription.
func handleDeleteSubscription(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	subID := vars["subscription"]
	c.Logger = c.Logger.WithField("subscription", subID)

	subscription, err := c.Store.GetSubscription(subID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query subscription")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if subscription == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if subscription.IsDeleted() {
		c.Logger.Warn("unable to delete subscription that is already deleted")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = c.Store.DeleteSubscription(subID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to mark subscription as deleted")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
