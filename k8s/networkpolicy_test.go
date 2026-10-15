// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	networkingv1 "k8s.io/api/networking/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
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

func TestUpdateNetworkPolicyIngressNamespaceSelector(t *testing.T) {
	testClient := newTestKubeClient()
	namespace := "testing"
	policy := &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: AllowMMExternal},
		Spec: networkingv1.NetworkPolicySpec{
			Ingress: []networkingv1.NetworkPolicyIngressRule{{
				From: []networkingv1.NetworkPolicyPeer{
					{NamespaceSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"name": "nginx"}}},
					{PodSelector: &metav1.LabelSelector{MatchLabels: map[string]string{"app": "mattermost"}}},
				},
			}},
		},
	}
	_, err := testClient.createOrUpdateNetworkPolicyV1(namespace, policy)
	require.NoError(t, err)

	t.Run("rewrites namespace selectors only", func(t *testing.T) {
		err := testClient.UpdateNetworkPolicyIngressNamespaceSelector(namespace, AllowMMExternal, "kube-system")
		require.NoError(t, err)

		updated, err := testClient.Clientset.NetworkingV1().NetworkPolicies(namespace).Get(context.TODO(), AllowMMExternal, metav1.GetOptions{})
		require.NoError(t, err)
		from := updated.Spec.Ingress[0].From
		require.Equal(t, map[string]string{"kubernetes.io/metadata.name": "kube-system"}, from[0].NamespaceSelector.MatchLabels)
		require.Nil(t, from[1].NamespaceSelector)
		require.Equal(t, map[string]string{"app": "mattermost"}, from[1].PodSelector.MatchLabels)
	})

	t.Run("missing policy", func(t *testing.T) {
		err := testClient.UpdateNetworkPolicyIngressNamespaceSelector(namespace, "missing", "kube-system")
		require.Error(t, err)
	})
}

func TestDeleteNetworkPolicy(t *testing.T) {
	testClient := newTestKubeClient()
	namespace := "testing"
	_, err := testClient.createOrUpdateNetworkPolicyV1(namespace, &networkingv1.NetworkPolicy{
		ObjectMeta: metav1.ObjectMeta{Name: DenyMetadataAccess},
	})
	require.NoError(t, err)

	require.NoError(t, testClient.DeleteNetworkPolicy(namespace, DenyMetadataAccess))
	_, err = testClient.Clientset.NetworkingV1().NetworkPolicies(namespace).Get(context.TODO(), DenyMetadataAccess, metav1.GetOptions{})
	require.True(t, k8sErrors.IsNotFound(err))

	t.Run("already deleted", func(t *testing.T) {
		require.NoError(t, testClient.DeleteNetworkPolicy(namespace, DenyMetadataAccess))
	})
}
