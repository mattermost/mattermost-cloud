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

func newTestGateway(namespace, name string, addresses []interface{}) *unstructured.Unstructured {
	gateway := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.networking.k8s.io/v1",
			"kind":       "Gateway",
			"metadata": map[string]interface{}{
				"name":      name,
				"namespace": namespace,
			},
		},
	}
	if addresses != nil {
		gateway.Object["status"] = map[string]interface{}{"addresses": addresses}
	}
	return gateway
}

func TestGetGatewayHostname(t *testing.T) {
	namespace := "kube-system"

	testClient := newTestKubeClient()
	testClient.DynamicClient = dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{gatewayGVR: "GatewayList"})

	for _, gateway := range []*unstructured.Unstructured{
		newTestGateway(namespace, "hostname", []interface{}{
			map[string]interface{}{"type": "IPAddress", "value": "10.0.0.1"},
			map[string]interface{}{"type": "Hostname", "value": "gw.elb.us-east-1.amazonaws.com"},
		}),
		newTestGateway(namespace, "ip-only", []interface{}{
			map[string]interface{}{"value": "10.0.0.1"},
		}),
		newTestGateway(namespace, "no-status", nil),
	} {
		_, err := testClient.DynamicClient.Resource(gatewayGVR).Namespace(namespace).Create(context.TODO(), gateway, metav1.CreateOptions{})
		require.NoError(t, err)
	}

	t.Run("returns hostname address", func(t *testing.T) {
		hostname, err := testClient.GetGatewayHostname(namespace, "hostname")
		require.NoError(t, err)
		require.Equal(t, "gw.elb.us-east-1.amazonaws.com", hostname)
	})

	t.Run("ip addresses only", func(t *testing.T) {
		_, err := testClient.GetGatewayHostname(namespace, "ip-only")
		require.Error(t, err)
	})

	t.Run("no status yet", func(t *testing.T) {
		_, err := testClient.GetGatewayHostname(namespace, "no-status")
		require.Error(t, err)
	})

	t.Run("gateway not found", func(t *testing.T) {
		_, err := testClient.GetGatewayHostname(namespace, "missing")
		require.Error(t, err)
	})
}
