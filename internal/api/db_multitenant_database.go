// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/model"
)

// initMultitenantDatabases registers multitenant database endpoints on the given router.
func initMultitenantDatabases(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	MultitenantDatabasesRouter := apiRouter.PathPrefix("/multitenant_databases").Subrouter()
	MultitenantDatabasesRouter.Handle("", addContext(handleGetMultitenantDatabases)).Methods("GET")

	MultitenantDatabaseRouter := apiRouter.PathPrefix("/multitenant_database/{multitenant_database:[A-Za-z0-9]{26}}").Subrouter()
	MultitenantDatabaseRouter.Handle("", addContext(handleGetMultitenantDatabase)).Methods("GET")
	MultitenantDatabaseRouter.Handle("", addContext(handleUpdateMultitenantDatabase)).Methods("PUT")
}

// handleGetMultitenantDatabases responds to GET /api/databases/multitenant_databases,
// returning a list of multitenant databases.
func handleGetMultitenantDatabases(c *Context, w http.ResponseWriter, r *http.Request) {
	paging, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filter := &model.MultitenantDatabaseFilter{
		VpcID:                 parseString(r.URL, "vpc_id", ""),
		DatabaseType:          parseString(r.URL, "database_type", ""),
		Paging:                paging,
		MaxInstallationsLimit: model.NoInstallationsLimit,
	}

	multitenantDatabases, err := c.Store.GetMultitenantDatabases(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query multitenant databases")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if multitenantDatabases == nil {
		multitenantDatabases = []*model.MultitenantDatabase{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, multitenantDatabases)
}

// handleGetMultitenantDatabase responds to GET /api/databases/multitenant_database/{multitenant_database},
// returning the multitenant database in question.
func handleGetMultitenantDatabase(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	multitenantDatabaseID := vars["multitenant_database"]
	c.Logger = c.Logger.WithField("multitenant_database", multitenantDatabaseID)

	multitenantDatabase, err := c.Store.GetMultitenantDatabase(multitenantDatabaseID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query multitenant database")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if multitenantDatabase == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, multitenantDatabase)
}

// handleUpdateMultitenantDatabase responds to PUT /api/databases/multitenant_database/{multitenant_database},
// updating the database configuration values.
func handleUpdateMultitenantDatabase(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	multitenantDatabaseID := vars["multitenant_database"]
	c.Logger = c.Logger.WithField("multitenant_database", multitenantDatabaseID)

	patchDatabaseRequest, err := model.NewPatchMultitenantDatabaseRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	multitenantDatabase, status, unlockOnce := lockDatabase(c, multitenantDatabaseID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if patchDatabaseRequest.Apply(multitenantDatabase) {
		err = c.Store.UpdateMultitenantDatabase(multitenantDatabase)
		if err != nil {
			c.Logger.WithError(err).Error("failed to update multitenant database")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	unlockOnce()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, multitenantDatabase)
}
