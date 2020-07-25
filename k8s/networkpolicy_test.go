// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"testing"

	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNetworkPolicyV1(t *testing.T) {
	testClient := newTestKubeClient()
	networkPolicy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: "test-networkPolicy"},
	}
	namespace := "testing"

	t.Run("create NetworkPolicy", func(t *testing.T) {
		result, err := testClient.createOrUpdateNetworkPolicyV1(namespace, networkPolicy)
		require.NoError(t, err)
		require.Equal(t, networkPolicy.GetName(), result.GetName())
	})
	t.Run("create duplicate NetworkPolicy", func(t *testing.T) {
		result, err := testClient.createOrUpdateNetworkPolicyV1(namespace, networkPolicy)
		require.NoError(t, err)
		require.Equal(t, networkPolicy.GetName(), result.GetName())
	})
}
