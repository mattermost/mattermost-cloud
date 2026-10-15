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

// ApplyCiliumMetadataDenyPolicy creates or updates a CiliumNetworkPolicy in the given
// namespace that blocks Mattermost pods from reaching the AWS instance metadata service
// (169.254.169.254). Used on Cilium clusters instead of the standard NetworkPolicy, which
// has policyTypes:[Egress] and causes Cilium to default-deny all egress including DNS.
func (kc *KubeClient) ApplyCiliumMetadataDenyPolicy(namespace string) error {
	policy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cilium.io/v2",
			"kind":       "CiliumNetworkPolicy",
			"metadata": map[string]interface{}{
				"name":      DenyMetadataAccess,
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"endpointSelector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": "mattermost",
					},
				},
				// Cilium changes enforcement mode for an endpoint when any
				// CiliumNetworkPolicy applies to it, even egress-only ones.
				// Explicitly allow the Cilium Gateway to reach Mattermost on port
				// 8065. Gateway traffic is proxied by the Cilium agent's Envoy and
				// carries the reserved "ingress" identity rather than a pod
				// identity. Allowing only that entity keeps other workloads from
				// connecting to Mattermost directly; Cilium adds allow rules
				// together, so a wider entity such as "cluster" would bypass the
				// namespace isolation in the standard NetworkPolicies.
				"ingress": []interface{}{
					map[string]interface{}{
						"fromEntities": []interface{}{"ingress"},
						"toPorts": []interface{}{
							map[string]interface{}{
								"ports": []interface{}{
									map[string]interface{}{
										"port":     "8065",
										"protocol": "TCP",
									},
								},
							},
						},
					},
				},
				// Allow all egress so Cilium does not default-deny in-cluster
				// traffic (DNS, PgBouncer, etc.) when this policy is applied.
				"egress": []interface{}{
					map[string]interface{}{
						"toEntities": []interface{}{"all"},
					},
				},
				// Explicitly deny the AWS instance metadata service.
				// egressDeny takes precedence over egress allow rules in Cilium.
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

	existing, err := client.Get(ctx, DenyMetadataAccess, metav1.GetOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		_, err = client.Create(ctx, policy, metav1.CreateOptions{})
		return err
	}
	if err != nil {
		return err
	}

	policy.SetResourceVersion(existing.GetResourceVersion())
	_, err = client.Update(ctx, policy, metav1.UpdateOptions{})
	return err
}
