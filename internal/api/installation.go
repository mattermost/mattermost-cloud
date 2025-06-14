// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/common"
	"github.com/mattermost/mattermost-cloud/internal/events"

	"github.com/pkg/errors"

	"github.com/mattermost/mattermost-cloud/internal/store"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/model"
)

// initInstallation registers installation endpoints on the given router.
func initInstallation(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	installationsRouter := apiRouter.PathPrefix("/installations").Subrouter()
	initInstallationBackup(installationsRouter, context)
	initInstallationRestoration(installationsRouter, context)
	initInstallationDBMigration(installationsRouter, context)

	installationsRouter.Handle("", addContext(handleGetInstallations)).Methods("GET")
	installationsRouter.Handle("", addContext(handleCreateInstallation)).Methods("POST")
	installationsRouter.Handle("/count", addContext(handleGetNumberOfInstallations)).Methods("GET")
	installationsRouter.Handle("/status", addContext(handleGetInstallationsStatus)).Methods("GET")

	installationRouter := apiRouter.PathPrefix("/installation/{installation:[A-Za-z0-9]{26}}").Subrouter()
	installationRouter.Handle("", addContext(handleGetInstallation)).Methods("GET")
	installationRouter.Handle("", addContext(handleRetryCreateInstallation)).Methods("POST")
	installationRouter.Handle("/mattermost", addContext(handleUpdateInstallation)).Methods("PUT")
	installationRouter.Handle("/volumes", addContext(handleCreateInstallationVolume)).Methods("POST")
	installationRouter.Handle("/volume/{volume-name}", addContext(handleUpdateInstallationVolume)).Methods("PUT")
	installationRouter.Handle("/volume/{volume-name}", addContext(handleDeleteInstallationVolume)).Methods("DELETE")
	installationRouter.Handle("/group/{group}", addContext(handleJoinGroup)).Methods("PUT")
	installationRouter.Handle("/group/{group}", addContext(handleAssignGroup)).Methods("POST")
	installationRouter.Handle("/group", addContext(handleLeaveGroup)).Methods("DELETE")
	installationRouter.Handle("/hibernate", addContext(handleHibernateInstallation)).Methods("POST")
	installationRouter.Handle("/wakeup", addContext(handleWakeupInstallation)).Methods("POST")
	installationRouter.Handle("", addContext(handleDeleteInstallation)).Methods("DELETE")
	installationRouter.Handle("/deletion", addContext(handleUpdateInstallationDeletion)).Methods("PUT")
	installationRouter.Handle("/deletion/cancel", addContext(handleCancelInstallationDeletion)).Methods("POST")
	installationRouter.Handle("/deletion/schedule", addContext(handleUpdateInstallationScheduledDeletion)).Methods("PUT")
	installationRouter.Handle("/annotations", addContext(handleAddInstallationAnnotations)).Methods("POST")
	installationRouter.Handle("/annotation/{annotation-name}", addContext(handleDeleteInstallationAnnotation)).Methods("DELETE")

	// DNS manipulation
	installationRouter.Handle("/dns", addContext(handleAddDNSRecord)).Methods("POST")
	installationRouter.Handle("/dns/{installationDNS}/set-primary", addContext(handleSetDomainNamePrimary)).Methods("POST")
	installationRouter.Handle("/dns/{installationDNS}", addContext(handleDeleteDNSRecord)).Methods("DELETE")
}

// handleGetInstallation responds to GET /api/installation/{installation}, returning the installation in question.
func handleGetInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	includeGroupConfig, includeGroupConfigOverrides, err := parseGroupConfig(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse group config parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	installation, err := c.Store.GetInstallationDTO(installationID, includeGroupConfig, includeGroupConfigOverrides)
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

	paging, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	includeGroupConfig, includeGroupConfigOverrides, err := parseGroupConfig(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse group parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	deletionLocked, err := parseDeletionLocked(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse deletion_locked parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filter := &model.InstallationFilter{
		OwnerID:        r.URL.Query().Get("owner"),
		GroupID:        r.URL.Query().Get("group"),
		State:          r.URL.Query().Get("state"),
		DNS:            r.URL.Query().Get("dns_name"),
		Name:           r.URL.Query().Get("name"),
		DeletionLocked: deletionLocked,
		Paging:         paging,
	}

	installations, err := c.Store.GetInstallationDTOs(filter, includeGroupConfig, includeGroupConfigOverrides)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query installations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if installations == nil {
		installations = []*model.InstallationDTO{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, installations)
}

// handlerGetNumberOfInstallations responds to GET /api/installations/count, returning the
// number of non-deleted installations
func handleGetNumberOfInstallations(c *Context, w http.ResponseWriter, r *http.Request) {
	includeDeleted, err := parseBool(r.URL, "include_deleted", false)
	if err != nil {
		includeDeleted = false
	}
	installationsCount, err := c.Store.GetInstallationsCount(&model.InstallationFilter{
		Paging: model.AllPages(includeDeleted),
	})
	if err != nil {
		c.Logger.WithError(err).Error("failed to query the number of installations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	result := model.InstallationsCount{Count: installationsCount}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, result)
}

// handleGetInstallationsStatus responds to GET /api/installations/status,
// returning the status of all non-deleted installations
func handleGetInstallationsStatus(c *Context, w http.ResponseWriter, r *http.Request) {
	installationsStatus, err := c.Store.GetInstallationsStatus()
	if err != nil {
		c.Logger.WithError(err).Error("failed to query for installations status")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, installationsStatus)
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

	if createInstallationRequest.GroupID == "" && len(createInstallationRequest.GroupSelectionAnnotations) > 0 {
		var groupID string
		groupID, err = selectGroupForAnnotation(c, createInstallationRequest.GroupSelectionAnnotations)
		if err != nil {
			c.Logger.WithError(err).Error("Failed to select group based on annotations")
			w.WriteHeader(common.ErrToStatus(err))
			return
		}
		createInstallationRequest.GroupID = groupID
	}

	var group *model.GroupDTO
	groupUnlockOnce := func() {}
	if len(createInstallationRequest.GroupID) != 0 {
		var status int
		group, status, groupUnlockOnce = lockGroup(c, createInstallationRequest.GroupID)
		if status != 0 {
			w.WriteHeader(status)
			return
		}
		defer groupUnlockOnce()
		if group.IsDeleted() {
			c.Logger.Errorf("cannot join installation to deleted group %s", createInstallationRequest.GroupID)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	if createInstallationRequest.Database == model.InstallationDatabaseExternal {
		err = c.AwsClient.SecretsManagerValidateExternalDatabaseSecret(createInstallationRequest.ExternalDatabaseConfig.SecretName)
		if err != nil {
			c.Logger.WithError(err).Error("failed to validate external secret")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	installation := model.Installation{
		Name:                       createInstallationRequest.Name,
		OwnerID:                    createInstallationRequest.OwnerID,
		GroupID:                    &createInstallationRequest.GroupID,
		Version:                    createInstallationRequest.Version,
		Image:                      createInstallationRequest.Image,
		Database:                   createInstallationRequest.Database,
		Filestore:                  createInstallationRequest.Filestore,
		License:                    createInstallationRequest.License,
		Size:                       createInstallationRequest.Size,
		Affinity:                   createInstallationRequest.Affinity,
		APISecurityLock:            createInstallationRequest.APISecurityLock,
		MattermostEnv:              createInstallationRequest.MattermostEnv,
		PriorityEnv:                createInstallationRequest.PriorityEnv,
		ScheduledDeletionTime:      createInstallationRequest.ScheduledDeletionTime,
		SingleTenantDatabaseConfig: createInstallationRequest.SingleTenantDatabaseConfig.ToDBConfig(createInstallationRequest.Database),
		ExternalDatabaseConfig:     createInstallationRequest.ExternalDatabaseConfig.ToDBConfig(createInstallationRequest.Database),
		PodProbeOverrides:          createInstallationRequest.PodProbeOverrides,
		CRVersion:                  model.DefaultCRVersion,
		State:                      model.InstallationStateCreationRequested,
	}

	dnsRecords := make([]*model.InstallationDNS, 0, len(createInstallationRequest.DNSNames))
	for _, domainName := range createInstallationRequest.DNSNames {
		dnsRecords = append(dnsRecords, &model.InstallationDNS{DomainName: domainName})
	}
	// Set first DNS record as primary
	dnsRecords[0].IsPrimary = true

	annotations, err := model.AnnotationsFromStringSlice(createInstallationRequest.Annotations)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to validate extra annotations")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = c.Store.CreateInstallation(&installation, annotations, dnsRecords)
	if err != nil {
		var uniqueErr *store.UniqueConstraintError
		if errors.As(err, &uniqueErr) {
			c.Logger.WithError(err).Error("domain name already in use")
			w.WriteHeader(http.StatusConflict)
			return
		}

		c.Logger.WithError(err).Error("failed to create installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = c.EventProducer.ProduceInstallationStateChangeEvent(&installation, model.NonApplicableState)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to create installation state change event")
	}

	groupUnlockOnce()
	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, installation.ToDTO(annotations, dnsRecords, nil))
}

func selectGroupForAnnotation(c *Context, annotations []string) (string, error) {
	groupAnnotations, err := c.Store.GetAnnotationsByName(annotations)
	if err != nil {
		return "", common.ErrWrap(http.StatusInternalServerError, err, "failed get annotations by name")
	}
	if len(groupAnnotations) != len(annotations) {
		return "", common.NewErr(http.StatusBadRequest, errors.New("some annotations for group selection do no exist"))
	}

	groups, err := c.Store.GetGroupDTOs(&model.GroupFilter{
		Paging: model.AllPagesNotDeleted(),
		Annotations: &model.AnnotationsFilter{
			MatchAllIDs: model.GetAnnotationsIDs(groupAnnotations),
		},
		WithInstallationCount: true,
	})
	if err != nil {
		return "", common.ErrWrap(http.StatusInternalServerError, err, "failed to get groups with annotations")
	}
	if len(groups) == 0 {
		return "", common.NewErr(http.StatusBadRequest, errors.New("no group matching all annotations found"))
	}

	// Select the group with less total installations
	var selectedGroup *model.GroupDTO
	for _, g := range groups {
		if selectedGroup == nil || g.GetInstallationCount() < selectedGroup.GetInstallationCount() {
			selectedGroup = g
		}
	}

	return selectedGroup.ID, nil
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

	newState := model.InstallationStateCreationRequested

	installationDTO, status, unlockOnce := getInstallationForTransition(c, installationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	err := updateInstallationState(c, installationDTO, newState)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to update installation state to %q", newState)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Notify even if we didn't make changes, to expedite even the no-op operations above.
	unlockOnce()
	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, installationDTO)
}

// handleUpdateInstallation responds to PUT /api/installation/{installation}/mattermost,
// updating the installation to the Mattermost configuration embedded in the request.
func handleUpdateInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	patchInstallationRequest, err := model.NewPatchInstallationRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	newState := model.InstallationStateUpdateRequested

	installationDTO, status, unlockOnce := getInstallationForTransition(c, installationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	oldState := installationDTO.State

	if patchInstallationRequest.Apply(installationDTO.Installation) {
		installationDTO.State = newState

		err = c.Store.UpdateInstallation(installationDTO.Installation)
		if err != nil {
			c.Logger.WithError(err).Error("failed to update installation")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		err = c.EventProducer.ProduceInstallationStateChangeEvent(installationDTO.Installation, oldState)
		if err != nil {
			c.Logger.WithError(err).Error("Failed to create installation state change event")
		}
	}

	unlockOnce()
	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, installationDTO)
}

// handleUpdateInstallationVolume responds to POST /api/installation/{installation}/volumes,
// which adds a custom installation volume.
func handleCreateInstallationVolume(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	createVolumeRequest, err := model.NewCreateInstallationVolumeRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	secretBytes, err := json.Marshal(createVolumeRequest.Data)
	if err != nil {
		c.Logger.WithError(err).Error("failed to marshal secret bytes")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	newState := model.InstallationStateUpdateRequested

	installationDTO, status, unlockOnce := getInstallationForTransition(c, installationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	oldState := installationDTO.State

	err = createVolumeRequest.Apply(installationDTO.Installation)
	if err != nil {
		c.Logger.WithError(err).Error("failed to apply request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err = c.AwsClient.SecretsManagerCreateSecret(
		createVolumeRequest.Volume.BackingSecret,
		fmt.Sprintf("Installation %s secret %s", installationDTO.ID, createVolumeRequest.Name),
		secretBytes,
		c.Logger,
	)
	if err != nil {
		c.Logger.WithError(err).Error("failed to create installation secret manager secret")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	installationDTO.State = newState

	err = c.Store.UpdateInstallation(installationDTO.Installation)
	if err != nil {
		c.Logger.WithError(err).Error("failed to update installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = c.EventProducer.ProduceInstallationStateChangeEvent(installationDTO.Installation, oldState)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to create installation state change event")
	}

	unlockOnce()
	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, installationDTO)
}

// handleUpdateInstallationVolume responds to PUT /api/installation/{installation}/volume/{volume-name},
// which updates a custom installation volume.
func handleUpdateInstallationVolume(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	volumeName := vars["volume-name"]
	c.Logger = c.Logger.WithField("installation", installationID)

	patchVolumeRequest, err := model.NewPatchInstallationVolumeRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	var secretBytes []byte
	if patchVolumeRequest.Data != nil {
		secretBytes, err = json.Marshal(patchVolumeRequest.Data)
		if err != nil {
			c.Logger.WithError(err).Error("failed to marshal secret bytes")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	newState := model.InstallationStateUpdateRequested

	installationDTO, status, unlockOnce := getInstallationForTransition(c, installationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	oldState := installationDTO.State

	secretName, err := patchVolumeRequest.Apply(installationDTO.Installation, volumeName)
	if err != nil {
		c.Logger.WithError(err).Error("failed to apply request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Only update the AWS secret if requested in the patch.
	if len(secretBytes) != 0 {
		err = c.AwsClient.SecretsManagerUpdateSecret(secretName, secretBytes, c.Logger)
		if err != nil {
			c.Logger.WithError(err).Error("failed to update installation secret manager secret")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	installationDTO.State = newState

	err = c.Store.UpdateInstallation(installationDTO.Installation)
	if err != nil {
		c.Logger.WithError(err).Error("failed to update installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = c.EventProducer.ProduceInstallationStateChangeEvent(installationDTO.Installation, oldState)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to create installation state change event")
	}

	unlockOnce()
	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, installationDTO)
}

// handleDeleteInstallationVolume responds to DELETE /api/installation/{installation}/volume/{volume-name},
// which deletes a custom installation volume.
func handleDeleteInstallationVolume(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	volumeName := vars["volume-name"]
	c.Logger = c.Logger.WithField("installation", installationID)

	newState := model.InstallationStateUpdateRequested

	installationDTO, status, unlockOnce := getInstallationForTransition(c, installationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	oldState := installationDTO.State

	volume, ok := (*installationDTO.Volumes)[volumeName]
	if !ok {
		c.Logger.Errorf("installation %s has no volume with name %s", installationID, volumeName)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := c.AwsClient.SecretsManagerEnsureSecretDeleted(volume.BackingSecret, c.Logger)
	if err != nil {
		c.Logger.WithError(err).Error("failed to delete installation secret manager secret")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Remove the volume. If this is the only installation volume then set the
	// VolumeMap back to nil.
	delete(*installationDTO.Volumes, volumeName)
	if len(*installationDTO.Volumes) == 0 {
		*installationDTO.Volumes = nil
	}

	installationDTO.State = newState

	err = c.Store.UpdateInstallation(installationDTO.Installation)
	if err != nil {
		c.Logger.WithError(err).Error("failed to update installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = c.EventProducer.ProduceInstallationStateChangeEvent(installationDTO.Installation, oldState)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to create installation state change event")
	}

	unlockOnce()
	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, installationDTO)
}

// handleJoinGroup responds to PUT /api/installation/{installation}/group/{group}, joining the group.
func handleJoinGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	groupID := vars["group"]
	c.Logger = c.Logger.WithField("installation", installationID)

	err := joinGroup(c, installationID, groupID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to join installation to the group")
		w.WriteHeader(common.ErrToStatus(err))
		return
	}

	c.Supervisor.Do()

	w.WriteHeader(http.StatusOK)
}

// handleAssignGroup responds to POST /api/installation/{installation}/group/assign,
// removing installation from current group (if any) and joining new one based to annotations.
func handleAssignGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	assignRequest, err := model.NewAssignInstallationGroupRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("invalid assign group request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	if len(assignRequest.GroupSelectionAnnotations) == 0 {
		c.Logger.Error("no annotations provided")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	groupID, err := selectGroupForAnnotation(c, assignRequest.GroupSelectionAnnotations)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to select group based on annotations")
		w.WriteHeader(common.ErrToStatus(err))
		return
	}
	c.Logger = c.Logger.WithField("group", groupID)
	c.Logger.Info("Selected group for Installation based on annotations. Joining the group...")

	err = joinGroup(c, installationID, groupID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to join installation to assigned group")
		w.WriteHeader(common.ErrToStatus(err))
		return
	}

	c.Supervisor.Do()

	w.WriteHeader(http.StatusOK)
}

// handleLeaveGroup responds to DELETE /api/installation/{installation}/group,
// leaving any existing group.
func handleLeaveGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	retainConfig, err := parseBool(r.URL, "retain_config", true)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse retain_config setting")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	newState := model.InstallationStateUpdateRequested

	// TODO: does it make sense to enforce normal update-requested valid states?
	// Should there be more or less valid states? Review this when necessary.
	installationDTO, status, unlockOnce := getInstallationForTransition(c, installationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if installationDTO.GroupID != nil {
		installationDTO.State = newState
		installationDTO.GroupID = nil
		installationDTO.GroupSequence = nil

		if retainConfig {
			// The installation is leaving the group, but the config is being set
			// to the group-merged version used while it was in the group. To do
			// so, we will get a merged copy of the installation out and will
			// manually update the necessary values.
			mergedInstallation, err := c.Store.GetInstallation(installationID, true, false)
			if err != nil {
				c.Logger.WithError(err).Error("failed to get group-merged installation")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}

			installationDTO.Version = mergedInstallation.Version
			installationDTO.Image = mergedInstallation.Image
			installationDTO.MattermostEnv = mergedInstallation.MattermostEnv
		}

		err := c.Store.UpdateInstallation(installationDTO.Installation)
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

// handleHibernateInstallation responds to POST /api/installation/{installation}/hibernate,
// moving the installation into a hibernation state.
func handleHibernateInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	newState := model.InstallationStateHibernationRequested

	installationDTO, status, unlockOnce := getInstallationForTransition(c, installationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	err := updateInstallationState(c, installationDTO, newState)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to update installation state to %q", newState)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	unlockOnce()
	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, installationDTO)
}

// handleWakeupInstallation responds to POST /api/installation/{installation}/wakeup,
// moving the installation out of a hibernation state.
func handleWakeupInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	patchInstallationRequest, err := model.NewPatchInstallationRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	newState := model.InstallationStateWakeUpRequested

	installationDTO, status, unlockOnce := getInstallationForTransition(c, installationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	oldState := installationDTO.State

	patchInstallationRequest.Apply(installationDTO.Installation)
	installationDTO.State = newState
	err = c.Store.UpdateInstallation(installationDTO.Installation)
	if err != nil {
		c.Logger.WithError(err).Error("failed to update installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = c.EventProducer.ProduceInstallationStateChangeEvent(installationDTO.Installation, oldState)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to create installation state change event")
	}

	unlockOnce()
	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, installationDTO)
}

// handleDeleteInstallation responds to DELETE /api/installation/{installation},
// beginning the process of deleting the installation.
func handleDeleteInstallation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	// Some installation states, such as creation failures, allow for going
	// directly to deletion-requested as they can't be properly put in the
	// state for pending deletion and there should be no possibility for data
	// loss. Check if that is the case before proceeding.
	newState := model.InstallationStateDeletionRequested
	installationDTO, err := c.Store.GetInstallationDTO(installationID, false, false)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if installationDTO == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if installationDTO.DeletionLocked {
		c.Logger.WithField("deletion-lock-conflict", "installation").Warn("Attempt to delete a deletion-locked installation")
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if !installationDTO.ValidTransitionState(newState) {
		// Fall back to the deletion pending state.
		newState = model.InstallationStateDeletionPendingRequested
	}

	installationDTO, status, unlockOnce := getInstallationForTransition(c, installationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	runningBackups, err := c.Store.GetInstallationBackups(&model.InstallationBackupFilter{
		InstallationID: installationID,
		States:         model.AllInstallationBackupsStatesRunning,
		Paging:         model.AllPagesNotDeleted(),
	})
	if err != nil {
		c.Logger.WithError(err).Error("failed to get list of running backups")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(runningBackups) > 0 {
		c.Logger.Error("there are running backups for the installation, cannot delete")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if newState == model.InstallationStateDeletionPendingRequested {
		// Set DeletionPendingExpiry to now+InstallationDeletionExpiryDefault
		installationDTO.DeletionPendingExpiry = model.GetMillisAtTime(time.Now().Add(c.InstallationDeletionExpiryDefault))
	}

	oldState := installationDTO.State
	installationDTO.State = newState
	err = c.Store.UpdateInstallation(installationDTO.Installation)
	if err != nil {
		c.Logger.WithError(err).Error("failed to update installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	// Try to parse the user ID from the request context, so it can be passed along with the webhook event
	actorID := ""
	if value := r.Context().Value(ContextKeyUserID{}); value != nil {
		if str, ok := value.(string); ok && str != "" {
			actorID = str
		}
	}

	err = c.EventProducer.ProduceInstallationStateChangeEvent(installationDTO.Installation, oldState, events.DataField{Key: "actor_id", Value: actorID})
	if err != nil {
		c.Logger.WithError(err).Error("Failed to create installation state change event")
	}

	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
}

// handleUpdateInstallationDeletion responds to PUT /api/installation/{installation}/deletion,
// beginning the process of updating the configuration of an upcoming deletion.
func handleUpdateInstallationDeletion(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	patchInstallationDeletionRequest, err := model.NewPatchInstallationDeletionRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// State won't change for this type of update.
	newState := model.InstallationStateDeletionPending

	installationDTO, status, unlockOnce := getInstallationForTransition(c, installationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if patchInstallationDeletionRequest.Apply(installationDTO.Installation) {
		err := c.Store.UpdateInstallation(installationDTO.Installation)
		if err != nil {
			c.Logger.WithError(err).Error("failed to update installation")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, installationDTO)
}

// handleCancelInstallationDeletion responds to POST /api/installation/{installation}/deletion/cancel,
// beginning the process of cancelling the pending deletion of the installation.
func handleCancelInstallationDeletion(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	newState := model.InstallationStateDeletionCancellationRequested

	installationDTO, status, unlockOnce := getInstallationForTransition(c, installationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	// If the installation is scheduled for deletion, cancel the scheduled
	// deletion so that it isn't immediately deleted again.
	if installationDTO.ScheduledDeletionTime != 0 {
		c.Logger.Debug("Removing scheduled deletion time as part of deletion cancellation")
		installationDTO.ScheduledDeletionTime = 0
	}
	installationDTO.State = newState

	err := c.Store.UpdateInstallation(installationDTO.Installation)
	if err != nil {
		c.Logger.WithError(err).Error("failed to update installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
}

// handleUpdateInstallationScheduledDeletion responds to PUT /api/installation/{installation}/deletion/schedule,
// updating the scheduled deletion time of an installation.
func handleUpdateInstallationScheduledDeletion(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID)

	patchRequest, err := model.NewPatchInstallationScheduledDeletionRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	installationDTO, status, unlockOnce := lockInstallation(c, installationID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if installationDTO.APISecurityLock {
		logSecurityLockConflict("installation", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if slices.Contains(model.AllInstallationDeletionStates, installationDTO.State) {
		c.Logger.Warnf("installation is in a deletion state, cannot update scheduled deletion time")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if patchRequest.Apply(installationDTO.Installation) {
		err := c.Store.UpdateInstallation(installationDTO.Installation)
		if err != nil {
			c.Logger.WithError(err).Error("failed to update installation")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, installationDTO)
}

// handleAddInstallationAnnotations responds to POST /api/installation/{installation}/annotations,
// adds the set of annotations to the Installation.
func handleAddInstallationAnnotations(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	c.Logger = c.Logger.WithField("installation", installationID).WithField("action", "add-installation-annotations")

	installationDTO, status, unlockOnce := lockInstallation(c, installationID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if installationDTO.APISecurityLock {
		logSecurityLockConflict("installation", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	annotations, err := annotationsFromRequest(r)
	if err != nil {
		c.Logger.WithError(err).Error("failed to get annotations from request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	annotations, err = c.Store.CreateInstallationAnnotations(installationID, annotations)
	if err != nil {
		c.Logger.WithError(err).Error("failed to create installation annotations")
		if errors.Is(err, store.ErrInstallationAnnotationDoNotMatchClusters) {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	installationDTO.Annotations = append(installationDTO.Annotations, annotations...)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, installationDTO)
}

// handleDeleteInstallationAnnotation responds to DELETE /api/installation/{installation}/annotation/{annotation-name},
// removes annotation from the Installation.
func handleDeleteInstallationAnnotation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	installationID := vars["installation"]
	annotationName := vars["annotation-name"]
	c.Logger = c.Logger.
		WithField("installation", installationID).
		WithField("action", "delete-installation-annotation").
		WithField("annotation-name", annotationName)

	installationDTO, status, unlockOnce := lockInstallation(c, installationID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if installationDTO.APISecurityLock {
		logSecurityLockConflict("installation", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err := c.Store.DeleteInstallationAnnotation(installationID, annotationName)
	if err != nil {
		c.Logger.WithError(err).Error("failed delete installation annotation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func joinGroup(c *Context, installationID, groupID string) error {
	installationDTO, status, installationUnlockOnce := lockInstallation(c, installationID)
	if status != 0 {
		return common.NewErr(status, errors.New("failed to lock installation"))
	}
	defer installationUnlockOnce()

	if installationDTO.APISecurityLock {
		logSecurityLockConflict("installation", c.Logger)
		return common.NewErr(http.StatusForbidden, errors.New("installation API locked"))
	}

	group, status, groupUnlockOnce := lockGroup(c, groupID)
	if status != 0 {
		return common.NewErr(status, errors.New("failed to lock group"))
	}
	defer groupUnlockOnce()
	if group.IsDeleted() {
		return common.NewErr(http.StatusBadRequest, errors.Errorf("cannot join installation to deleted group %s", groupID))
	}

	// Update the installation, but don't directly modify the configuration.
	// The supervisor will manage this later.
	if installationDTO.GroupID == nil || *installationDTO.GroupID != groupID {
		installationDTO.GroupID = &groupID

		err := c.Store.UpdateInstallation(installationDTO.Installation)
		if err != nil {
			return common.NewErr(http.StatusInternalServerError, errors.Wrap(err, "failed to update installation"))
		}
	}

	installationUnlockOnce()
	groupUnlockOnce()
	return nil
}

// updateInstallationState updates installation state in database and sends appropriate webhook.
func updateInstallationState(c *Context, installationDTO *model.InstallationDTO, newState string) error {
	if installationDTO.State == newState {
		return nil
	}

	oldState := installationDTO.State
	installationDTO.State = newState

	err := c.Store.UpdateInstallationState(installationDTO.Installation)
	if err != nil {
		return err
	}

	err = c.EventProducer.ProduceInstallationStateChangeEvent(installationDTO.Installation, oldState)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to create installation state change event")
	}

	return nil
}

// getInstallationForTransition locks the installation and validates if it can be transitioned to desired state.
func getInstallationForTransition(c *Context, installationID, newState string) (*model.InstallationDTO, int, func()) {
	installationDTO, status, unlockOnce := lockInstallation(c, installationID)
	if status != 0 {
		c.Logger.Errorf("Failed to lock installation, status: %d", status)
		return nil, status, unlockOnce
	}

	if installationDTO.APISecurityLock {
		unlockOnce()
		logSecurityLockConflict("installation", c.Logger)
		return nil, http.StatusForbidden, unlockOnce
	}

	if !installationDTO.ValidTransitionState(newState) {
		unlockOnce()
		c.Logger.Warnf("unable to transition installation to %q while in state %q", newState, installationDTO.State)
		return nil, http.StatusBadRequest, unlockOnce
	}

	return installationDTO, 0, unlockOnce
}
