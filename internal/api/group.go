package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/model"
)

// initGroup registers group endpoints on the given router.
func initGroup(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc) *contextHandler {
		return newContextHandler(context, handler)
	}

	groupsRouter := apiRouter.PathPrefix("/groups").Subrouter()
	groupsRouter.Handle("", addContext(handleGetGroups)).Methods("GET")
	groupsRouter.Handle("", addContext(handleCreateGroup)).Methods("POST")

	groupRouter := apiRouter.PathPrefix("/group/{group:[A-Za-z0-9]{26}}").Subrouter()
	groupRouter.Handle("", addContext(handleGetGroup)).Methods("GET")
	groupRouter.Handle("", addContext(handleUpdateGroup)).Methods("PUT")
	groupRouter.Handle("", addContext(handleDeleteGroup)).Methods("DELETE")
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

	outputJSON(c, w, groups)
}

// handleCreateGroup responds to POST /api/groups, beginning the process of creating a new group.
func handleCreateGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	createGroupRequest, err := newCreateGroupRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	group := model.Group{
		Name:        createGroupRequest.Name,
		Description: createGroupRequest.Description,
		Version:     createGroupRequest.Version,
	}

	err = c.Store.CreateGroup(&group)
	if err != nil {
		c.Logger.WithError(err).Error("failed to create group")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Supervisor.Do()

	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, group)
}

// handleGetGroup responds to GET /api/groups/{group}, returning the group in question.
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

	outputJSON(c, w, group)
}

// handleUpdateGroup responds to PUT /api/groups/{group}, updating the group.
func handleUpdateGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["group"]
	c.Logger = c.Logger.WithField("group", groupID)

	patchGroupRequest, err := newPatchGroupRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	group, err := c.Store.GetGroup(groupID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to fetch group")
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if group == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	if patchGroupRequest.Apply(group) {
		err := c.Store.UpdateGroup(group)
		if err != nil {
			c.Logger.WithError(err).Error("failed to update group")
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
	}

	c.Supervisor.Do()

	w.WriteHeader(http.StatusOK)
}

// handleDeleteGroup responds to DELETE /api/groups/{group}, marking the group as deleted.
//
// Installations will not automatically leave the group, but they will no longer consider the
// group version as an upgrade target.
func handleDeleteGroup(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	groupID := vars["group"]
	c.Logger = c.Logger.WithField("group", groupID)

	group, err := c.Store.GetGroup(groupID)
	if err != nil {
		c.Logger.WithError(err).Error("failed to fetch group")
		w.WriteHeader(http.StatusInternalServerError)
		return
	} else if group == nil {
		w.WriteHeader(http.StatusNotFound)
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
