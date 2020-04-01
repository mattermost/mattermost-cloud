package api

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"
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
	installationRouter.Handle("/mattermost", addContext(handleUpdateInstallation)).Methods("PUT")
	installationRouter.Handle("/group/{group}", addContext(handleJoinGroup)).Methods("PUT")
	installationRouter.Handle("/group", addContext(handleLeaveGroup)).Methods("DELETE")
	installationRouter.Handle("", addContext(handleDeleteInstallation)).Methods("DELETE")
}

// handleGetInstallation responds to GET /api/installation/{installation}, returning the installation in question.
func handleGetInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	includeGroupConfig, includeGroupConfigOverrides, err := parseGroupConfig(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	installation, err := c.Store.GetInstallation(installationID, includeGroupConfig, includeGroupConfigOverrides)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if installation == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, installation)
}

// handleGetInstallations responds to GET /api/installations, returning the specified page of installations.
func handleGetInstallations(c *Context, w http.ResponseWriter, r *http.Request) {
	var err error
	owner := r.URL.Query().Get("owner")
	group := r.URL.Query().Get("group")

	page, perPage, includeDeleted, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	includeGroupConfig, includeGroupConfigOverrides, err := parseGroupConfig(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filter := &model.InstallationFilter{
		OwnerID:        owner,
		GroupID:        group,
		Page:           page,
		PerPage:        perPage,
		IncludeDeleted: includeDeleted,
	}

	installations, err := c.Store.GetInstallations(filter, includeGroupConfig, includeGroupConfigOverrides)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query installations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if installations == nil {
		installations = []*model.Installation{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, installations)
}

// handleCreateInstallation responds to POST /api/installations, beginning the process of creating
// a new installation.
func handleCreateInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	createInstallationRequest, err := model.NewCreateInstallationRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if len(createInstallationRequest.GroupID) != 0 {
		group, err := c.Store.GetGroup(createInstallationRequest.GroupID)
		if err != nil {
			c.Logger.WithError(err).Error("failed to query group")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if group == nil {
			w.WriteHeader(http.StatusNotFound)
			return
		}
	}

	installation := model.Installation{
		OwnerID:       createInstallationRequest.OwnerID,
		GroupID:       &createInstallationRequest.GroupID,
		Version:       createInstallationRequest.Version,
		Image:         createInstallationRequest.Image,
		DNS:           createInstallationRequest.DNS,
		Database:      createInstallationRequest.Database,
		Filestore:     createInstallationRequest.Filestore,
		License:       createInstallationRequest.License,
		Size:          createInstallationRequest.Size,
		Affinity:      createInstallationRequest.Affinity,
		MattermostEnv: createInstallationRequest.MattermostEnv,
		State:         model.InstallationStateCreationRequested,
	}

	err = c.Store.CreateInstallation(&installation)
	if err != nil {
		c.Logger.WithError(err).Error("failed to create installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallation,
		ID:        installation.ID,
		NewState:  model.InstallationStateCreationRequested,
		OldState:  "n/a",
		Timestamp: time.Now().UnixNano(),
	}
	err = webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		c.Logger.WithError(err).Error("Unable to process and send webhooks")
	}

	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
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

	newState := model.InstallationStateCreationRequested

	if !installation.ValidTransitionState(newState) {
		c.Logger.Warnf("unable to retry installation creation while in state %s", installation.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if installation.State != newState {
		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeInstallation,
			ID:        installation.ID,
			NewState:  newState,
			OldState:  installation.State,
			Timestamp: time.Now().UnixNano(),
		}
		installation.State = newState

		err := c.Store.UpdateInstallation(installation)
		if err != nil {
			c.Logger.WithError(err).Errorf("failed to retry installation creation")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState))
		if err != nil {
			c.Logger.WithError(err).Error("Unable to process and send webhooks")
		}
	}

	// Notify even if we didn't make changes, to expedite even the no-op operations above.
	unlockOnce()
	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, installation)
}

// handleUpdateInstallation responds to PUT /api/installation/{installation}/mattermost,
// updating the installation to the Mattermost configuration embedded in the request.
func handleUpdateInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	updateInstallationRequest, err := model.NewUpdateInstallationRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	installation, status, unlockOnce := lockInstallation(c, installationID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	newState := model.InstallationStateUpdateRequested

	if !installation.ValidTransitionState(newState) {
		c.Logger.Warnf("unable to update installation while in state %s", installation.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallation,
		ID:        installation.ID,
		NewState:  newState,
		OldState:  installation.State,
		Timestamp: time.Now().UnixNano(),
	}
	installation.State = newState
	installation.Version = updateInstallationRequest.Version
	installation.License = updateInstallationRequest.License
	installation.Image = updateInstallationRequest.Image
	installation.MattermostEnv = updateInstallationRequest.MattermostEnv

	err = c.Store.UpdateInstallation(installation)
	if err != nil {
		c.Logger.WithError(err).Error("failed to mark installation for update")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		c.Logger.WithError(err).Error("Unable to process and send webhooks")
	}

	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
}

// handleJoinGroup responds to PUT /api/installation/{installation}/group/{group}, joining the group.
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

// handleLeaveGroup responds to DELETE /api/installation/{installation}/group, leaving any existing group.
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

// handleDeleteInstallation responds to DELETE /api/installation/{installation}, beginning the process of
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

	newState := model.InstallationStateDeletionRequested

	if !installation.ValidTransitionState(newState) {
		c.Logger.Warnf("unable to delete installation while in state %s", installation.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if installation.State != newState {
		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeInstallation,
			ID:        installation.ID,
			NewState:  newState,
			OldState:  installation.State,
			Timestamp: time.Now().UnixNano(),
		}
		installation.State = newState

		err := c.Store.UpdateInstallation(installation)
		if err != nil {
			c.Logger.WithError(err).Error("failed to mark installation for deletion")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState))
		if err != nil {
			c.Logger.WithError(err).Error("Unable to process and send webhooks")
		}
	}

	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
}
