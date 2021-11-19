// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewSubscriptionFromReader(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		subscription, err := NewSubscriptionFromReader(bytes.NewReader([]byte(
			"",
		)))
		require.NoError(t, err)
		require.Equal(t, &Subscription{}, subscription)
	})

	t.Run("invalid", func(t *testing.T) {
		subscription, err := NewSubscriptionFromReader(bytes.NewReader([]byte(
			"{test",
		)))
		require.Error(t, err)
		require.Nil(t, subscription)
	})

	t.Run("valid", func(t *testing.T) {
		subscription, err := NewSubscriptionFromReader(bytes.NewReader([]byte(
			`{"id":"abcd", "name":"test", "url":"http://events", "ownerid":"owner", "eventType":"event", "lastDeliveryStatus":"fail", "lastDeliveryAttemptAt":100, "failureThreshold":200, "createAt":300, "deleteAt":400}`,
		)))
		require.NoError(t, err)
		require.Equal(t, &Subscription{
			ID:                    "abcd",
			Name:                  "test",
			URL:                   "http://events",
			OwnerID:               "owner",
			EventType:             "event",
			LastDeliveryStatus:    "fail",
			LastDeliveryAttemptAt: 100,
			FailureThreshold:      200,
			CreateAt:              300,
			DeleteAt:              400,
		}, subscription)
	})
}

func TestNewSubscriptionsFromReader(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		subscriptions, err := NewSubscriptionsFromReader(bytes.NewReader([]byte(
			"",
		)))
		require.NoError(t, err)
		require.Equal(t, []*Subscription{}, subscriptions)
	})

	t.Run("invalid", func(t *testing.T) {
		subscriptions, err := NewSubscriptionsFromReader(bytes.NewReader([]byte(
			"{test",
		)))
		require.Error(t, err)
		require.Nil(t, subscriptions)
	})

	t.Run("valid", func(t *testing.T) {
		subscriptions, err := NewSubscriptionsFromReader(bytes.NewReader([]byte(
			`[{"id":"abcd"},{"id":"efgh"}]`,
		)))
		require.NoError(t, err)
		require.Equal(t, []*Subscription{
			{ID: "abcd"},
			{ID: "efgh"},
		}, subscriptions)
	})
}
