package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/model"
)

// initInstallation registers installation endpoints on the given router.
func initInstallation(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	installationsRouter := apiRouter.PathPrefix("/installations").Subrouter()
	installationsRouter.Handle("", addContext(handleGetInstallations)).Methods("GET")
	installationsRouter.Handle("", addContext(handleCreateInstallation)).Methods("POST")

	installationRouter := apiRouter.PathPrefix("/installation/{installation:[A-Za-z0-9]{26}}").Subrouter()
	installationRouter.Handle("", addContext(handleGetInstallation)).Methods("GET")
	installationRouter.Handle("", addContext(handleRetryCreateInstallation)).Methods("POST")
	installationRouter.Handle("/mattermost/{version}", addContext(handleUpgradeInstallation)).Methods("PUT")
	installationRouter.Handle("/group/{group}", addContext(handleJoinGroup)).Methods("PUT")
	installationRouter.Handle("/group", addContext(handleLeaveGroup)).Methods("DELETE")
	installationRouter.Handle("", addContext(handleDeleteInstallation)).Methods("DELETE")
}

// handleGetInstallations responds to GET /api/installations, returning the specified page of installations.
func handleGetInstallations(c *Context, w http.ResponseWriter, r *http.Request) {
	var err error
	owner := r.URL.Query().Get("owner")

	page, perPage, includeDeleted, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filter := &model.InstallationFilter{
		OwnerID:        owner,
		Page:           page,
		PerPage:        perPage,
		IncludeDeleted: includeDeleted,
	}

	installations, err := c.Store.GetInstallations(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query installations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if installations == nil {
		installations = []*model.Installation{}
	}

	outputJSON(c, w, installations)
}

// handleCreateInstallation responds to POST /api/installations, beginning the process of creating
// a new installation.
func handleCreateInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	createInstallationRequest, err := newCreateInstallationRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	installation := model.Installation{
		OwnerID:  createInstallationRequest.OwnerID,
		Version:  createInstallationRequest.Version,
		DNS:      createInstallationRequest.DNS,
		Affinity: createInstallationRequest.Affinity,
		State:    model.InstallationStateCreationRequested,
	}

	err = c.Store.CreateInstallation(&installation)
	if err != nil {
		c.Logger.WithError(err).Error("failed to create installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, installation)
}

// handleRetryCreateInstallation responds to POST /api/installation/{installation}, retrying a
// previously failed creation.
//
// Note that other operations on a installation may be retried by simply repeating the same request,
// but repeating handleCreateInstallation would create a second installation.
func handleRetryCreateInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	installation, status, unlockOnce := lockInstallation(c, installationID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	switch installation.State {
	case model.InstallationStateCreationRequested:
	case model.InstallationStateCreationFailed:
	default:
		c.Logger.Warnf("unable to retry installation creation while in state %s", installation.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if installation.State != model.InstallationStateCreationRequested {
		installation.State = model.InstallationStateCreationRequested

		err := c.Store.UpdateInstallation(installation)
		if err != nil {
			c.Logger.WithError(err).Errorf("failed to retry installation creation")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	// Notify even if we didn't make changes, to expedite even the no-op operations above.
	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, installation)
}

// handleGetInstallation responds to GET /api/installations/{installation}, returning the installation in question.
func handleGetInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	installation, err := c.Store.GetInstallation(installationID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if installation == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	outputJSON(c, w, installation)
}

// handleUpgradeInstallation responds to PUT /api/installations/{installation}/mattermost/{version}, upgrading
// the installation to the given Mattermost version.
func handleUpgradeInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	version := vars["version"]
	c.Logger = c.Logger.WithField("installation", installationID)

	installation, status, unlockOnce := lockInstallation(c, installationID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	switch installation.State {
	case model.InstallationStateStable:
	case model.InstallationStateUpgradeRequested:
	case model.InstallationStateUpgradeFailed:
	default:
		c.Logger.Warnf("unable to upgrade installation while in state %s", installation.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// TODO: Support something other than "latest".
	if version != "latest" {
		c.Logger.Warnf("unsupported mattermost version %s", version)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if installation.State != model.InstallationStateUpgradeRequested {
		installation.State = model.InstallationStateUpgradeRequested

		err := c.Store.UpdateInstallation(installation)
		if err != nil {
			c.Logger.WithError(err).Error("failed to mark installation for upgrade")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
}

// handleJoinGroup responds to PUT /api/installations/{installation}/group/{group}, joining the group.
func handleJoinGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	groupID := vars["group"]
	c.Logger = c.Logger.WithField("installation", installationID)

	installation, status, unlockOnce := lockInstallation(c, installationID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	group, err := c.Store.GetGroup(groupID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query group")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if group == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	// Update the group, but don't directly modify the version. The supervisor
	// will transition the installation to the appropriate version.
	if installation.GroupID == nil || *installation.GroupID != groupID {
		installation.GroupID = &groupID

		err := c.Store.UpdateInstallation(installation)
		if err != nil {
			c.Logger.WithError(err).Error("failed to update installation")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusOK)
}

// handleLeaveGroup responds to DELETE /api/installations/{installation}/group, leaving any existing group.
func handleLeaveGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	installation, status, unlockOnce := lockInstallation(c, installationID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if installation.GroupID != nil {
		installation.GroupID = nil

		err := c.Store.UpdateInstallation(installation)
		if err != nil {
			c.Logger.WithError(err).Error("failed to update installation")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusOK)
}

// handleDeleteInstallation responds to DELETE /api/installations/{installation}, beginning the process of
// deleting the installation.
func handleDeleteInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	installation, status, unlockOnce := lockInstallation(c, installationID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	switch installation.State {
	case model.InstallationStateStable:
	case model.InstallationStateCreationRequested:
	case model.InstallationStateCreationFailed:
	case model.InstallationStateDeletionRequested:
	case model.InstallationStateDeletionInProgress:
	case model.InstallationStateDeletionFailed:
	default:
		c.Logger.Warnf("unable to delete installation while in state %s", installation.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if installation.State != model.InstallationStateDeletionRequested {
		installation.State = model.InstallationStateDeletionRequested

		err := c.Store.UpdateInstallation(installation)
		if err != nil {
			c.Logger.WithError(err).Error("failed to mark installation for deletion")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
}
