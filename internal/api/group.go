// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/model"
)

// initGroup registers group endpoints on the given router.
func initGroup(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	groupsRouter := apiRouter.PathPrefix("/groups").Subrouter()
	groupsRouter.Handle("", addContext(handleGetGroups)).Methods("GET")
	groupsRouter.Handle("", addContext(handleCreateGroup)).Methods("POST")
	groupsRouter.Handle("/status", addContext(handleGetGroupsStatus)).Methods("GET")

	groupRouter := apiRouter.PathPrefix("/group/{group:[A-Za-z0-9]{26}}").Subrouter()
	groupRouter.Handle("", addContext(handleGetGroup)).Methods("GET")
	groupRouter.Handle("", addContext(handleUpdateGroup)).Methods("PUT")
	groupRouter.Handle("", addContext(handleDeleteGroup)).Methods("DELETE")
	groupRouter.Handle("/status", addContext(handleGetGroupStatus)).Methods("GET")
}

// handleGetGroup responds to GET /api/group/{group}, returning the group in question.
func handleGetGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["group"]
	c.Logger = c.Logger.WithField("group", groupID)

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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, group)
}

// handleGetGroups responds to GET /api/groups, returning the specified page of groups.
func handleGetGroups(c *Context, w http.ResponseWriter, r *http.Request) {
	page, perPage, includeDeleted, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	filter := &model.GroupFilter{
		Page:           page,
		PerPage:        perPage,
		IncludeDeleted: includeDeleted,
	}

	groups, err := c.Store.GetGroups(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query groups")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if groups == nil {
		groups = []*model.Group{}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, groups)
}

// handleCreateGroup responds to POST /api/groups, beginning the process of creating a new group.
func handleCreateGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	createGroupRequest, err := model.NewCreateGroupRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	group := model.Group{
		Name:            createGroupRequest.Name,
		Description:     createGroupRequest.Description,
		Version:         createGroupRequest.Version,
		Image:           createGroupRequest.Image,
		MaxRolling:      createGroupRequest.MaxRolling,
		APISecurityLock: createGroupRequest.APISecurityLock,
		MattermostEnv:   createGroupRequest.MattermostEnv,
	}

	err = c.Store.CreateGroup(&group)
	if err != nil {
		c.Logger.WithError(err).Error("failed to create group")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, group)
}

// handleUpdateGroup responds to PUT /api/group/{group}, updating the group.
func handleUpdateGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["group"]
	c.Logger = c.Logger.WithField("group", groupID)

	patchGroupRequest, err := model.NewPatchGroupRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	group, status, unlockOnce := lockGroup(c, groupID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if group.APISecurityLock {
		logSecurityLockConflict("group", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	if patchGroupRequest.Apply(group) {
		err := c.Store.UpdateGroup(group, patchGroupRequest.ForceSequenceUpdate)
		if err != nil {
			c.Logger.WithError(err).Error("failed to update group")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	c.Supervisor.Do()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, group)
}

// handleDeleteGroup responds to DELETE /api/group/{group}, marking the group as deleted.
//
// The group must contain no installations in order to be deleted.
func handleDeleteGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["group"]
	c.Logger = c.Logger.WithField("group", groupID)

	group, status, unlockOnce := lockGroup(c, groupID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if group.APISecurityLock {
		logSecurityLockConflict("group", c.Logger)
		w.WriteHeader(http.StatusForbidden)
		return
	}

	installations, err := c.Store.GetInstallations(&model.InstallationFilter{
		GroupID:        groupID,
		Page:           0,
		PerPage:        model.AllPerPage,
		IncludeDeleted: false,
	}, false, false)
	if err != nil {
		c.Logger.WithError(err).Error("failed to get installations in group")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(installations) != 0 {
		c.Logger.Errorf("unable to delete group while it still has %d installation members", len(installations))
		w.WriteHeader(http.StatusForbidden)
		return
	}

	err = c.Store.DeleteGroup(group.ID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to mark group for deletion")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Supervisor.Do()

	w.WriteHeader(http.StatusOK)
}

// handleGetGroupStatus responds to GET /api/group/{group}/status,
// returning the rollout status of the group in question.
func handleGetGroupStatus(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["group"]
	c.Logger = c.Logger.WithField("group", groupID)

	groupStatus, err := c.Store.GetGroupStatus(groupID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query group status")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if groupStatus == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, groupStatus)
}

// handleGetGroupsStatus responds to GET /api/groups/status,
// returning the rollout status of all groups.
func handleGetGroupsStatus(c *Context, w http.ResponseWriter, r *http.Request) {
	filter := &model.GroupFilter{
		PerPage:        model.AllPerPage,
		IncludeDeleted: false,
	}

	groups, err := c.Store.GetGroups(filter)
	if err != nil {
		c.Logger.WithError(err).Error("failed to query groups")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if groups == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	var groupsStatus []model.GroupsStatus

	for _, group := range groups {
		var groupStatus model.GroupsStatus
		status, err := c.Store.GetGroupStatus(group.ID)
		if err != nil {
			c.Logger.WithError(err).Error("failed to query group status")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		if status == nil {
			continue
		}

		groupStatus.ID = group.ID
		groupStatus.Status = *status

		groupsStatus = append(groupsStatus, groupStatus)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, groupsStatus)
}
