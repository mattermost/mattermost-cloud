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
	return kc.applyOrUpdateCiliumPolicy(ctx, namespace, "deny-metadata-access", policy)
}

// ApplyCiliumGatewayIngressPolicy creates or updates a CiliumNetworkPolicy that allows
// ingress from the host entity on port 8065. This is required on Cilium-based clusters
// because the Cilium Gateway API envoy runs with hostNetwork:true and appears as the
// "host" identity — standard NetworkPolicy namespace selectors never match it.
func (kc *KubeClient) ApplyCiliumGatewayIngressPolicy(namespace string) error {
	policy := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "cilium.io/v2",
			"kind":       "CiliumNetworkPolicy",
			"metadata": map[string]interface{}{
				"name":      "external-mm-cilium-allow",
				"namespace": namespace,
			},
			"spec": map[string]interface{}{
				"endpointSelector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": "mattermost",
					},
				},
				"ingress": []interface{}{
					map[string]interface{}{
						"fromEntities": []interface{}{"host"},
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
			},
		},
	}

	ctx := context.TODO()
	return kc.applyOrUpdateCiliumPolicy(ctx, namespace, "external-mm-cilium-allow", policy)
}

func (kc *KubeClient) applyOrUpdateCiliumPolicy(ctx context.Context, namespace, name string, policy *unstructured.Unstructured) error {
	client := kc.DynamicClient.Resource(ciliumNetworkPolicyGVR).Namespace(namespace)

	existing, err := client.Get(ctx, name, metav1.GetOptions{})
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
