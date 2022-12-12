// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"github.com/gorilla/mux"
)

// initDatabases registers database endpoints on the given router.
func initDatabases(apiRouter *mux.Router, context *Context) {
	databasesRouter := apiRouter.PathPrefix("/databases").Subrouter()
	initMultitenantDatabases(databasesRouter, context)
	initLogicalDatabases(databasesRouter, context)
	initDatabaseSchemas(databasesRouter, context)
}
