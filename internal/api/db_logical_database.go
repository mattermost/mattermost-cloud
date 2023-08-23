// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/model"
)

// initLogicalDatabases registers logical database endpoints on the given router.
func initLogicalDatabases(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	LogicalDatabasesRouter := apiRouter.PathPrefix("/logical_databases").Subrouter()
	LogicalDatabasesRouter.Handle("", addContext(handleGetLogicalDatabases)).Methods("GET")

	LogicalDatabaseRouter := apiRouter.PathPrefix("/logical_database/{logical_database:[A-Za-z0-9]{26}}").Subrouter()
	LogicalDatabaseRouter.Handle("", addContext(handleGetLogicalDatabase)).Methods("GET")
	LogicalDatabaseRouter.Handle("", addContext(handleDeleteLogicalDatabase)).Methods("DELETE")
}

// handleGetLogicalDatabases responds to GET /api/databases/logical_databases,
// returning a list of logical databases.
func handleGetLogicalDatabases(c *Context, w http.ResponseWriter, r *http.Request) {
	paging, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filter := &model.LogicalDatabaseFilter{
		MultitenantDatabaseID: parseString(r.URL, "multitenant_database_id", ""),
		Paging:                paging,
	}

	logicalDatabases, err := c.Store.GetLogicalDatabases(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query logical databases")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if logicalDatabases == nil {
		logicalDatabases = []*model.LogicalDatabase{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, logicalDatabases)
}

// handleGetLogicalDatabase responds to GET /api/databases/logical_database/{logical_database},
// returning the logical database in question.
func handleGetLogicalDatabase(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	logicalDatabaseID := vars["logical_database"]
	c.Logger = c.Logger.WithField("logical_database", logicalDatabaseID)

	logicalDatabase, err := c.Store.GetLogicalDatabase(logicalDatabaseID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query logical database")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if logicalDatabase == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, logicalDatabase)
}

// handleDeleteLogicalDatabase responds to DELETE /api/databases/logical_database/{logical_database},
// deleting an empty logical database.
func handleDeleteLogicalDatabase(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	logicalDatabaseID := vars["logical_database"]
	c.Logger = c.Logger.WithField("logical_database", logicalDatabaseID)

	logicalDatabase, err := c.Store.GetLogicalDatabase(logicalDatabaseID)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to get logical database by ID")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if logicalDatabase == nil {
		c.Logger.Debug("Logical database for deletion not found")
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if logicalDatabase.DeleteAt > 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	multitenantDatabase, status, unlockOnce := lockDatabase(c, logicalDatabase.MultitenantDatabaseID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	schemas, err := c.Store.GetDatabaseSchemas(&model.DatabaseSchemaFilter{
		LogicalDatabaseID: logicalDatabase.ID,
		Paging:            model.AllPagesNotDeleted(),
	})
	if err != nil {
		c.Logger.WithError(err).Error("Failed to get database schemas")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(schemas) > 0 {
		c.Logger.Error("Cannot delete logical database that still contains active schemas")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = c.AwsClient.DeletePGBouncerLogicalDatabase(multitenantDatabase, logicalDatabase.Name, c.Logger)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to delete pgbouncer logical database")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = c.Store.DeleteLogicalDatabase(logicalDatabase.ID)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to mark logical database as deleted")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Logger.Infof("Deleted logical database %s from shared database %s", logicalDatabase.ID, multitenantDatabase.ID)

	w.WriteHeader(http.StatusNoContent)
}
