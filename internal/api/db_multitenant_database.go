// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"
	"strconv"

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
	MultitenantDatabaseRouter.Handle("", addContext(handleDeleteMultitenantDatabase)).Methods("DELETE")
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

// handleDeleteMultitenantDatabase responds to DELETE /api/databases/multitenant_database/{multitenant_database},
// marking the database as deleted.
// WARNING: It does not delete actual database cluster.
func handleDeleteMultitenantDatabase(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	multitenantDatabaseID := vars["multitenant_database"]
	c.Logger = c.Logger.WithField("multitenant_database", multitenantDatabaseID)

	query := r.URL.Query()
	force, err := strconv.ParseBool(query.Get("force"))
	if err != nil { // If we failed to pase, assume false
		force = false
	}

	db, err := c.Store.GetMultitenantDatabase(multitenantDatabaseID)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to get multitenant database by ID")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if db == nil {
		c.Logger.Debug("Multitenat database for deletion not found")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if db.DeleteAt > 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// If deletion is forced we do not check and just delete.
	if !force {
		var exists bool
		exists, err = c.AwsClient.RDSDBCLusterExists(db.RdsClusterID)
		if err != nil {
			c.Logger.WithError(err).Error("Failed to check if DB cluster exists")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if exists {
			c.Logger.Error("Cannot delete multitenant database if DB cluster exists.")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	err = c.Store.DeleteMultitenantDatabase(multitenantDatabaseID)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to mark multitenant database as deleted")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
