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

	groupRouter := apiRouter.PathPrefix("/group/{group:[A-Za-z0-9]{26}}").Subrouter()
	groupRouter.Handle("", addContext(handleGetGroup)).Methods("GET")
	groupRouter.Handle("", addContext(handleUpdateGroup)).Methods("PUT")
	groupRouter.Handle("", addContext(handleDeleteGroup)).Methods("DELETE")

	groupRouter.Handle("/mattermost/{version:[A-Za-z0-9.]+}", addContext(handleChangeGroupMattermostVersion)).Methods("PUT")
}

// handleChangeGroupMattermostVersion responds to PUT
// /api/group/{group}/mattermost/{version}, upgrading or downgrading
// the installation to the Mattermost version embedded in the request.
func handleChangeGroupMattermostVersion(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	groupID, ok := vars["group"]
	if !ok {
		c.Logger.Error("failed to get group ID from request parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	c.Logger = c.Logger.WithField("group", groupID)

	version, ok := vars["version"]
	if !ok {
		c.Logger.Error("failed to get version from request parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	c.Logger = c.Logger.WithField("version", version)

	group, err := c.Store.GetGroup(groupID)
	if err != nil {
		c.Logger.WithError(err).Errorf("failed to find group with ID %s", groupID)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if group == nil {
		c.Logger.Errorf("failed to find group with ID %s", groupID)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	group.Version = version
	err = c.Store.UpdateGroup(group)
	if err != nil {
		c.Logger.WithError(err).Errorf("error writing group object to database %#v", group)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
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

	err = createGroupRequest.Validate()
	if err != nil {
		c.Logger.WithError(err).Error("invalid create group request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	group := model.Group{
		Name:          createGroupRequest.Name,
		Description:   createGroupRequest.Description,
		Version:       createGroupRequest.Version,
		MattermostEnv: createGroupRequest.MattermostEnv,
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

	err = patchGroupRequest.MattermostEnv.Validate()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		c.Logger.WithError(err).Error("invalid env var settings")
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

// handleDeleteGroup responds to DELETE /api/group/{group}, marking the group as deleted.
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

	installations, err := c.Store.GetInstallations(&model.InstallationFilter{
		GroupID:        groupID,
		Page:           0,
		PerPage:        model.AllPerPage,
		IncludeDeleted: false,
	})
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
