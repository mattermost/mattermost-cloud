// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"github.com/gorilla/mux"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Register registers the API endpoints on the given router.
func Register(rootRouter *mux.Router, context *Context) {
	// metrics handler at /metrics
	rootRouter.Handle("/metrics", promhttp.Handler())

	// api handler at /api
	apiRouter := rootRouter.PathPrefix("/api").Subrouter()
	initCluster(apiRouter, context)
	initInstallation(apiRouter, context)
	initClusterInstallation(apiRouter, context)
	initGroup(apiRouter, context)
	initWebhook(apiRouter, context)
	initDatabases(apiRouter, context)
	initSecurity(apiRouter, context)
}
