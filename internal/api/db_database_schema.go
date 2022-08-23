// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/model"
)

// initDatabaseSchemas registers database schema endpoints on the given router.
func initDatabaseSchemas(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc, name string) *contextHandler {
		return newContextHandler(context, handler, name)
	}

	DatabaseSchemasRouter := apiRouter.PathPrefix("/database_schemas").Subrouter()
	DatabaseSchemasRouter.Handle("", addContext(handleGetDatabaseSchemas, "handleGetDatabaseSchemas")).Methods("GET")

	DatabaseSchemaRouter := apiRouter.PathPrefix("/database_schema/{database_schema:[A-Za-z0-9]{26}}").Subrouter()
	DatabaseSchemaRouter.Handle("", addContext(handleGetDatabaseSchema, "handleGetDatabaseSchema")).Methods("GET")
}

// handleGetDatabaseSchemas responds to GET /api/databases/database_schemas,
// returning a list of database schemas.
func handleGetDatabaseSchemas(c *Context, w http.ResponseWriter, r *http.Request) {
	paging, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filter := &model.DatabaseSchemaFilter{
		LogicalDatabaseID: parseString(r.URL, "logical_database_id", ""),
		InstallationID:    parseString(r.URL, "installation_id", ""),
		Paging:            paging,
	}

	databaseSchemas, err := c.Store.GetDatabaseSchemas(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query database schemas")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if databaseSchemas == nil {
		databaseSchemas = []*model.DatabaseSchema{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, databaseSchemas)
}

// handleGetDatabaseSchema responds to GET /api/databases/database_schemas/{database_schema},
// returning the database schema in question.
func handleGetDatabaseSchema(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	databaseSchemaID := vars["database_schema"]
	c.Logger = c.Logger.WithField("database_schema", databaseSchemaID)

	databaseSchema, err := c.Store.GetDatabaseSchema(databaseSchemaID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query database schema")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if databaseSchema == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, databaseSchema)
}
