// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func TestApplyCiliumMetadataDenyPolicy(t *testing.T) {
	namespace := "testing"
	testClient := newTestKubeClient()
	testClient.DynamicClient = dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{ciliumNetworkPolicyGVR: "CiliumNetworkPolicyList"})

	getPolicy := func(t *testing.T) *unstructured.Unstructured {
		policy, err := testClient.DynamicClient.Resource(ciliumNetworkPolicyGVR).Namespace(namespace).Get(context.TODO(), DenyMetadataAccess, metav1.GetOptions{})
		require.NoError(t, err)
		return policy
	}

	t.Run("create", func(t *testing.T) {
		require.NoError(t, testClient.ApplyCiliumMetadataDenyPolicy(namespace))
		policy := getPolicy(t)

		ingress, found, err := unstructured.NestedSlice(policy.Object, "spec", "ingress")
		require.NoError(t, err)
		require.True(t, found)
		require.Len(t, ingress, 1)
		entities, _, err := unstructured.NestedStringSlice(ingress[0].(map[string]interface{}), "fromEntities")
		require.NoError(t, err)
		require.Equal(t, []string{"ingress"}, entities)

		egressDeny, _, err := unstructured.NestedSlice(policy.Object, "spec", "egressDeny")
		require.NoError(t, err)
		require.Len(t, egressDeny, 1)
		cidrs, _, err := unstructured.NestedStringSlice(egressDeny[0].(map[string]interface{}), "toCIDR")
		require.NoError(t, err)
		require.Equal(t, []string{"169.254.169.254/32"}, cidrs)
	})

	t.Run("update existing policy", func(t *testing.T) {
		existing := getPolicy(t)
		require.NoError(t, unstructured.SetNestedField(existing.Object, []interface{}{}, "spec", "egressDeny"))
		_, err := testClient.DynamicClient.Resource(ciliumNetworkPolicyGVR).Namespace(namespace).Update(context.TODO(), existing, metav1.UpdateOptions{})
		require.NoError(t, err)

		require.NoError(t, testClient.ApplyCiliumMetadataDenyPolicy(namespace))
		egressDeny, _, err := unstructured.NestedSlice(getPolicy(t).Object, "spec", "egressDeny")
		require.NoError(t, err)
		require.Len(t, egressDeny, 1)
	})
}
