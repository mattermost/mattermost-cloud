package api

import "github.com/gorilla/mux"

// Register registers the API endpoints on the given router.
func Register(rootRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	apiRouter := rootRouter.PathPrefix("/api").Subrouter()

	clustersRouter := apiRouter.PathPrefix("/clusters").Subrouter()
	clustersRouter.Handle("", addContext(handleGetClusters)).Methods("GET")
	clustersRouter.Handle("", addContext(handleCreateCluster)).Methods("POST")

	clusterRouter := apiRouter.PathPrefix("/cluster/{cluster:[A-Za-z0-9]{26}}").Subrouter()
	clusterRouter.Handle("", addContext(handleGetCluster)).Methods("GET")
	clusterRouter.Handle("", addContext(handleRetryCreateCluster)).Methods("POST")
	clusterRouter.Handle("/kubernetes/{version}", addContext(handleUpgradeCluster)).Methods("PUT")
	clusterRouter.Handle("", addContext(handleDeleteCluster)).Methods("DELETE")
}
