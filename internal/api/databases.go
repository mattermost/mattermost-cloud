// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"github.com/gorilla/mux"
)

// initDatabases registers database endpoints on the given router.
func initDatabases(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	// TODO: retire these endpoints
	databasesRouter := apiRouter.PathPrefix("/databases").Subrouter()
	databasesRouter.Handle("", addContext(handleGetMultitenantDatabases)).Methods("GET")

	databaseRouter := apiRouter.PathPrefix("/database/{multitenant_database:[A-Za-z0-9]{26}}").Subrouter()
	databaseRouter.Handle("", addContext(handleUpdateMultitenantDatabase)).Methods("PUT")

	// Begin new endpoints
	initMultitenantDatabases(databasesRouter, context)
	initLogicalDatabases(databasesRouter, context)
	initDatabaseSchemas(databasesRouter, context)
}
