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
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateGetDeleteSubscriptions(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})

	ts := httptest.NewServer(router)
	client := model.NewClient(ts.URL)

	// Create subscription
	subRequest := &model.CreateSubscriptionRequest{
		Name:             "My sub",
		URL:              "https://test",
		OwnerID:          "tester",
		EventType:        model.ResourceStateChangeEventType,
		FailureThreshold: 10 * time.Minute,
	}
	sub, err := client.CreateSubscription(subRequest)
	require.NoError(t, err)
	assert.NotEmpty(t, sub.ID)
	assert.Equal(t, subRequest.Name, sub.Name)
	assert.Equal(t, subRequest.URL, sub.URL)
	assert.Equal(t, subRequest.OwnerID, sub.OwnerID)
	assert.Equal(t, subRequest.EventType, sub.EventType)
	assert.Equal(t, subRequest.FailureThreshold, sub.FailureThreshold)
	assert.Equal(t, int64(0), sub.DeleteAt)
	assert.NotEmpty(t, sub.CreateAt)

	// Get subscription
	fetchedSub, err := client.GetSubscription(sub.ID)
	require.NoError(t, err)
	assert.Equal(t, sub, fetchedSub)

	t.Run("should return 404 on not found", func(t *testing.T) {
		notFoundSub, err := client.GetSubscription(model.NewID())
		require.NoError(t, err)
		assert.Nil(t, notFoundSub)
	})

	// Delete subscription
	err = client.DeleteSubscription(sub.ID)
	require.NoError(t, err)

	t.Run("fail to delete twice", func(t *testing.T) {
		err = client.DeleteSubscription(sub.ID)
		require.Error(t, err)
	})

	fetchedSub, err = client.GetSubscription(sub.ID)
	require.NoError(t, err)
	assert.True(t, fetchedSub.DeleteAt > 0)
}

func TestListSubscriptions(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	router := mux.NewRouter()
	api.Register(router, &api.Context{
		Store:      sqlStore,
		Supervisor: &mockSupervisor{},
		Logger:     logger,
	})

	ts := httptest.NewServer(router)
	client := model.NewClient(ts.URL)

	// Create subscriptions
	subsRequests := []*model.CreateSubscriptionRequest{
		{OwnerID: "tester", EventType: model.ResourceStateChangeEventType},
		{OwnerID: "tester", EventType: "test"},
		{OwnerID: "other-tester", EventType: model.ResourceStateChangeEventType},
	}

	subs := []*model.Subscription{}

	for i := range subsRequests {
		newSub, err := client.CreateSubscription(subsRequests[i])
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
		subs = append(subs, newSub)
	}

	// Get subscriptions
	for _, testCase := range []struct {
		description string
		filter      model.ListSubscriptionsRequest
		found       []*model.Subscription
	}{
		{
			description: "all",
			filter:      model.ListSubscriptionsRequest{Paging: model.AllPagesWithDeleted()},
			found:       subs,
		},
		{
			description: "for owner",
			filter:      model.ListSubscriptionsRequest{Paging: model.AllPagesWithDeleted(), Owner: "tester"},
			found:       []*model.Subscription{subs[0], subs[1]},
		},
		{
			description: "for event type",
			filter:      model.ListSubscriptionsRequest{Paging: model.AllPagesWithDeleted(), EventType: model.ResourceStateChangeEventType},
			found:       []*model.Subscription{subs[0], subs[2]},
		},
		{
			description: "for owner and event type",
			filter:      model.ListSubscriptionsRequest{Paging: model.AllPagesWithDeleted(), Owner: "tester", EventType: model.ResourceStateChangeEventType},
			found:       []*model.Subscription{subs[0]},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {

			listedSubs, err := client.ListSubscriptions(&testCase.filter)
			require.NoError(t, err)
			require.Equal(t, len(testCase.found), len(listedSubs))

			for i := 0; i < len(testCase.found); i++ {
				assert.Equal(t, testCase.found[i], listedSubs[len(testCase.found)-1-i])
			}
		})
	}
}
