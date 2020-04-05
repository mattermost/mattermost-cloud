package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/model"
)

// initMigration registers migration endpoints on the given router.
func initClusterInstallationMigration(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	migrationsRouter := apiRouter.PathPrefix("/migrations").Subrouter()
	migrationsRouter.Handle("", addContext(handleCreateClusterInstallationMigration)).Methods("POST")
}

// handleCreateMigration responds to POST /api/migrations, beginning the process of creating
// a new migration.
func handleCreateClusterInstallationMigration(c *Context, w http.ResponseWriter, r *http.Request) {
	createMigrationRequest, err := model.NewCreateClusterInstallationMigrationRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	migration := model.ClusterInstallationMigration{
		ClusterID:             createMigrationRequest.ClusterID,
		ClusterInstallationID: createMigrationRequest.ClusterInstallationID,
		State:                 model.CIMigrationCreationRequested,
	}

	err = c.Store.CreateClusterInstallationMigration(&migration)
	if err != nil {
		c.Logger.WithError(err).Error("failed to create migration")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, migration)
}
