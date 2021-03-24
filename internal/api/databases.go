// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/model"
)

// initDatabases registers database endpoints on the given router.
func initDatabases(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	databaseRouter := apiRouter.PathPrefix("/databases").Subrouter()
	databaseRouter.Handle("", addContext(handleGetDatabases)).Methods("GET")
}

// handleGetDatabases responds to GET /api/databases, returning a list of
// multitenant databases.
func handleGetDatabases(c *Context, w http.ResponseWriter, r *http.Request) {
	paging, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	vpcID, databaseType := parseDatabaseListRequest(r.URL)

	filter := &model.MultitenantDatabaseFilter{
		VpcID:                 vpcID,
		DatabaseType:          databaseType,
		Paging:                paging,
		MaxInstallationsLimit: model.NoInstallationsLimit,
	}

	databases, err := c.Store.GetMultitenantDatabases(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query multitenant databases")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if databases == nil {
		databases = []*model.MultitenantDatabase{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, databases)
}
