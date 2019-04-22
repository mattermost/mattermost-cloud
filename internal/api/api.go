package api

import "github.com/gorilla/mux"

// Register registers the API endpoints on the given router.
func Register(rootRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	apiRouter := rootRouter.PathPrefix("/api").Subrouter()
	apiRouter.Handle("/clusters", addContext(handleGetClusters)).Methods("GET")
	apiRouter.Handle("/clusters", addContext(handleCreateCluster)).Methods("POST")
	apiRouter.Handle("/cluster/{cluster}", addContext(handleGetCluster)).Methods("GET")
	apiRouter.Handle("/cluster/{cluster}/kubernetes/{version}", addContext(handleUpgradeCluster)).Methods("PUT")
	apiRouter.Handle("/cluster/{cluster}", addContext(handleDeleteCluster)).Methods("DELETE")
}
