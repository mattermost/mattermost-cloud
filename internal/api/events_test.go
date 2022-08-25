// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api_test

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/api"
	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/internal/testutil"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListStateChangeEvents(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:         sqlStore,
		Supervisor:    &mockSupervisor{},
		EventProducer: testutil.SetupTestEventsProducer(sqlStore, logger),
		Metrics:       &mockMetrics{},
		Logger:        logger,
	})

	ts := httptest.NewServer(router)
	client := model.NewClient(ts.URL)
	model.SetDeployOperators(true, true)

	// Create Installation and Cluster
	installation, err := client.CreateInstallation(&model.CreateInstallationRequest{
		OwnerID: "test",
		DNS:     "test.com",
	})
	require.NoError(t, err)
	time.Sleep(1 * time.Millisecond)
	cluster, err := client.CreateCluster(&model.CreateClusterRequest{})
	require.NoError(t, err)

	// List Events
	eventsData, err := client.ListStateChangeEvents(&model.ListStateChangeEventsRequest{Paging: model.AllPagesNotDeleted()})
	require.NoError(t, err)

	assert.Equal(t, 2, len(eventsData))
	assert.Equal(t, cluster.ID, eventsData[0].StateChange.ResourceID)
	assert.Equal(t, model.TypeCluster, eventsData[0].StateChange.ResourceType)
	assert.Equal(t, "n/a", eventsData[0].StateChange.OldState)
	assert.Equal(t, cluster.State, eventsData[0].StateChange.NewState)
	assert.Equal(t, model.ResourceStateChangeEventType, eventsData[0].Event.EventType)

	assert.Equal(t, installation.ID, eventsData[1].StateChange.ResourceID)
	assert.Equal(t, model.TypeInstallation, eventsData[1].StateChange.ResourceType)
	assert.Equal(t, "n/a", eventsData[1].StateChange.OldState)
	assert.Equal(t, installation.State, eventsData[1].StateChange.NewState)
	assert.Equal(t, model.ResourceStateChangeEventType, eventsData[1].Event.EventType)

	// List by resource ID
	eventsData, err = client.ListStateChangeEvents(&model.ListStateChangeEventsRequest{ResourceID: installation.ID, Paging: model.AllPagesNotDeleted()})
	require.NoError(t, err)
	assert.Equal(t, 1, len(eventsData))
	assert.Equal(t, installation.ID, eventsData[0].StateChange.ResourceID)
	assert.Equal(t, model.TypeInstallation, eventsData[0].StateChange.ResourceType)

	// List by resource type
	eventsData, err = client.ListStateChangeEvents(&model.ListStateChangeEventsRequest{ResourceType: model.TypeCluster, Paging: model.AllPagesNotDeleted()})
	require.NoError(t, err)
	assert.Equal(t, 1, len(eventsData))
	assert.Equal(t, cluster.ID, eventsData[0].StateChange.ResourceID)
	assert.Equal(t, model.TypeCluster, eventsData[0].StateChange.ResourceType)
}
