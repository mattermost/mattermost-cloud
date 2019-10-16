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
	installationRouter.Handle("/mattermost", addContext(handleUpgradeInstallation)).Methods("PUT")
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

	w.Header().Set("Content-Type", "application/json")
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

	installation := model.Installation{
		OwnerID:   createInstallationRequest.OwnerID,
		Version:   createInstallationRequest.Version,
		DNS:       createInstallationRequest.DNS,
		Database:  createInstallationRequest.Database,
		Filestore: createInstallationRequest.Filestore,
		License:   createInstallationRequest.License,
		Size:      createInstallationRequest.Size,
		Affinity:  createInstallationRequest.Affinity,
		State:     model.InstallationStateCreationRequested,
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

	w.WriteHeader(http.StatusAccepted)
	w.Header().Set("Content-Type", "application/json")
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
		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeInstallation,
			ID:        installation.ID,
			NewState:  model.InstallationStateCreationRequested,
			OldState:  installation.State,
			Timestamp: time.Now().UnixNano(),
		}
		installation.State = model.InstallationStateCreationRequested

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

// handleGetInstallation responds to GET /api/installation/{installation}, returning the installation in question.
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

	w.Header().Set("Content-Type", "application/json")
	outputJSON(c, w, installation)
}

// handleUpgradeInstallation responds to PUT /api/installation/{installation}/mattermost, upgrading
// the installation to the Mattermost version embedded in the request.
func handleUpgradeInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	upgradeInstallationRequest, err := model.NewUpgradeInstallationRequestFromReader(r.Body)
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

	switch installation.State {
	case model.InstallationStateStable:
	case model.InstallationStateUpgradeRequested:
	case model.InstallationStateUpgradeInProgress:
	case model.InstallationStateUpgradeFailed:
	default:
		c.Logger.Warnf("unable to upgrade installation while in state %s", installation.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if installation.State != model.InstallationStateUpgradeRequested ||
		installation.Version != upgradeInstallationRequest.Version ||
		installation.License != upgradeInstallationRequest.License {
		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeInstallation,
			ID:        installation.ID,
			NewState:  model.InstallationStateUpgradeRequested,
			OldState:  installation.State,
			Timestamp: time.Now().UnixNano(),
		}
		installation.State = model.InstallationStateUpgradeRequested
		installation.Version = upgradeInstallationRequest.Version
		installation.License = upgradeInstallationRequest.License

		err := c.Store.UpdateInstallation(installation)
		if err != nil {
			c.Logger.WithError(err).Error("failed to mark installation for upgrade")
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

	switch installation.State {
	case model.InstallationStateStable:
	case model.InstallationStateCreationRequested:
	case model.InstallationStateCreationPreProvisioning:
	case model.InstallationStateCreationInProgress:
	case model.InstallationStateCreationDNS:
	case model.InstallationStateCreationNoCompatibleClusters:
	case model.InstallationStateCreationFailed:
	case model.InstallationStateUpgradeRequested:
	case model.InstallationStateUpgradeInProgress:
	case model.InstallationStateUpgradeFailed:
	case model.InstallationStateDeletionRequested:
	case model.InstallationStateDeletionInProgress:
	case model.InstallationStateDeletionFinalCleanup:
	case model.InstallationStateDeletionFailed:
	default:
		c.Logger.Warnf("unable to delete installation while in state %s", installation.State)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if installation.State != model.InstallationStateDeletionRequested {
		webhookPayload := &model.WebhookPayload{
			Type:      model.TypeInstallation,
			ID:        installation.ID,
			NewState:  model.InstallationStateDeletionRequested,
			OldState:  installation.State,
			Timestamp: time.Now().UnixNano(),
		}
		installation.State = model.InstallationStateDeletionRequested

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
