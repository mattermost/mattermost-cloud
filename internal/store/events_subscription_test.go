// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package store

import (
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountSubscriptionsForEvent(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	sub1 := &model.Subscription{
		EventType: model.ResourceStateChangeEventType,
	}
	err := sqlStore.CreateSubscription(sub1)
	require.NoError(t, err)
	sub2 := &model.Subscription{
		EventType: model.ResourceStateChangeEventType,
	}
	err = sqlStore.CreateSubscription(sub2)
	require.NoError(t, err)
	sub3 := &model.Subscription{
		EventType: "different",
	}
	err = sqlStore.CreateSubscription(sub3)
	require.NoError(t, err)

	subsCount, err := sqlStore.CountSubscriptionsForEvent(model.ResourceStateChangeEventType)
	require.NoError(t, err)
	assert.Equal(t, int64(2), subsCount)
}

func TestGetCreateUpdateSubscription(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	sub := &model.Subscription{
		Name:                  "test",
		URL:                   "http://test",
		OwnerID:               "tester",
		EventType:             model.ResourceStateChangeEventType,
		LastDeliveryStatus:    model.SubscriptionDeliverySucceeded,
		LastDeliveryAttemptAt: 100,
		FailureThreshold:      2 * time.Minute,
	}
	err := sqlStore.CreateSubscription(sub)
	require.NoError(t, err)
	assert.NotEmpty(t, sub.ID)

	fetchedSub, err := sqlStore.GetSubscription(sub.ID)
	require.NoError(t, err)

	assert.Equal(t, "test", fetchedSub.Name)
	assert.Equal(t, "http://test", fetchedSub.URL)
	assert.Equal(t, "tester", fetchedSub.OwnerID)
	assert.Equal(t, model.ResourceStateChangeEventType, fetchedSub.EventType)
	assert.Equal(t, model.SubscriptionDeliverySucceeded, fetchedSub.LastDeliveryStatus)
	assert.Equal(t, int64(100), fetchedSub.LastDeliveryAttemptAt)
	assert.Equal(t, 2*time.Minute, fetchedSub.FailureThreshold)

	t.Run("unknown ID", func(t *testing.T) {
		s, err2 := sqlStore.GetSubscription(model.NewID())
		require.NoError(t, err2)
		assert.Nil(t, s)
	})

	sub.LastDeliveryStatus = model.SubscriptionDeliveryFailed
	sub.LastDeliveryAttemptAt = 10000
	sub.Name = "should not change"

	err = sqlStore.UpdateSubscriptionStatus(sub)
	require.NoError(t, err)

	fetchedSub, err = sqlStore.GetSubscription(sub.ID)
	require.NoError(t, err)
	assert.Equal(t, model.SubscriptionDeliveryFailed, fetchedSub.LastDeliveryStatus)
	assert.Equal(t, int64(10000), fetchedSub.LastDeliveryAttemptAt)
	assert.Equal(t, "test", fetchedSub.Name)

	err = sqlStore.DeleteSubscription(sub.ID)
	require.NoError(t, err)

	fetchedSub, err = sqlStore.GetSubscription(sub.ID)
	require.NoError(t, err)
	assert.True(t, fetchedSub.DeleteAt > 0)
}

func TestGetSubscriptions(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := MakeTestSQLStore(t, logger)
	defer CloseConnection(t, sqlStore)

	subs := []*model.Subscription{
		{OwnerID: "tester1", EventType: model.ResourceStateChangeEventType},
		{OwnerID: "tester1", EventType: "test"},
		{OwnerID: "tester2", EventType: model.ResourceStateChangeEventType},
		{OwnerID: "tester3", EventType: "test2"},
	}

	for i := range subs {
		err := sqlStore.CreateSubscription(subs[i])
		require.NoError(t, err)
		time.Sleep(1 * time.Millisecond)
	}

	err := sqlStore.DeleteSubscription(subs[3].ID)
	require.NoError(t, err)

	for _, testCase := range []struct {
		description string
		filter      *model.SubscriptionsFilter
		fetchedIds  []string
	}{
		{
			description: "fetch all not deleted",
			filter:      &model.SubscriptionsFilter{Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{subs[2].ID, subs[1].ID, subs[0].ID},
		},
		{
			description: "fetch all with deleted",
			filter:      &model.SubscriptionsFilter{Paging: model.AllPagesWithDeleted()},
			fetchedIds:  []string{subs[3].ID, subs[2].ID, subs[1].ID, subs[0].ID},
		},
		{
			description: "fetch by owner",
			filter:      &model.SubscriptionsFilter{Owner: "tester1", Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{subs[1].ID, subs[0].ID},
		},
		{
			description: "fetch by event type",
			filter:      &model.SubscriptionsFilter{EventType: model.ResourceStateChangeEventType, Paging: model.AllPagesNotDeleted()},
			fetchedIds:  []string{subs[2].ID, subs[0].ID},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			fetchedSubs, err := sqlStore.GetSubscriptions(testCase.filter)
			require.NoError(t, err)
			assert.Equal(t, len(testCase.fetchedIds), len(fetchedSubs))

			for i, b := range fetchedSubs {
				assert.Equal(t, testCase.fetchedIds[i], b.ID)
			}
		})
	}
}
