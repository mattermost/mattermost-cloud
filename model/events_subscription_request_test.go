// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewCreateSubscriptionRequestFromReader(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		createSubscriptionRequest, err := NewCreateSubscriptionRequestFromReader(bytes.NewReader([]byte(
			"",
		)))
		require.NoError(t, err)
		require.Equal(t, &CreateSubscriptionRequest{}, createSubscriptionRequest)
	})

	t.Run("invalid", func(t *testing.T) {
		createSubscriptionRequest, err := NewCreateSubscriptionRequestFromReader(bytes.NewReader([]byte(
			"{test",
		)))
		require.Error(t, err)
		require.Nil(t, createSubscriptionRequest)
	})

	t.Run("valid", func(t *testing.T) {
		createSubscriptionRequest, err := NewCreateSubscriptionRequestFromReader(bytes.NewReader([]byte(
			`{"name":"test","url":"http://test", "ownerID":"owner","failureThreshold":100}`,
		)))
		require.NoError(t, err)
		require.Equal(t, &CreateSubscriptionRequest{
			Name:             "test",
			URL:              "http://test",
			OwnerID:          "owner",
			FailureThreshold: 100,
		}, createSubscriptionRequest)
	})
}
