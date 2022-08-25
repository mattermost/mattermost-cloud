// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/model"
)

// initEvent registers events endpoints on the given router.
func initEvent(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	eventRouter := apiRouter.PathPrefix("/events").Subrouter()

	stateChangeRouter := eventRouter.PathPrefix("/state_change").Subrouter()

	stateChangeRouter.Handle("", addContext(handleListStateChangeEvents)).Methods("GET")
}

// handleListEvents responds to GET /api/events/state_change, returning the specified page of subscriptions.
func handleListStateChangeEvents(c *Context, w http.ResponseWriter, r *http.Request) {
	paging, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filter := &model.StateChangeEventFilter{
		Paging:       paging,
		ResourceType: model.ResourceType(r.URL.Query().Get("resource_type")),
		ResourceID:   r.URL.Query().Get("resource_id"),
	}

	events, err := c.Store.GetStateChangeEvents(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query events")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if events == nil {
		events = []*model.StateChangeEventData{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, events)
}
