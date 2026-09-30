// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package k8s

import (
	"context"

	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var ciliumNetworkPolicyGVR = schema.GroupVersionResource{
	Group:    "cilium.io",
	Version:  "v2",
	Resource: "ciliumnetworkpolicies",
}

// ApplyCiliumMetadataDenyPolicy creates or updates a CiliumNetworkPolicy in the given namespace
// that blocks Mattermost pods from reaching the AWS instance metadata service (169.254.169.254).
// This is required on Cilium-based clusters where standard Kubernetes NetworkPolicy egress rules
// with ipBlock exceptions are not reliably enforced.
func (kc *KubeClient) ApplyCiliumMetadataDenyPolicy(namespace string) error {
	policy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cilium.io/v2",
			"kind":       "CiliumNetworkPolicy",
			"metadata": map[string]interface{}{
				"name":      "deny-metadata-access",
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"endpointSelector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": "mattermost",
					},
				},
				"egressDeny": []interface{}{
					map[string]interface{}{
						"toCIDR": []interface{}{"169.254.169.254/32"},
					},
				},
			},
		},
	}

	ctx := context.TODO()
	client := kc.DynamicClient.Resource(ciliumNetworkPolicyGVR).Namespace(namespace)

	_, err := client.Get(ctx, "deny-metadata-access", metav1.GetOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		_, err = client.Create(ctx, policy, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}

	_, err = client.Update(ctx, policy, metav1.UpdateOptions{})
	return err
}
