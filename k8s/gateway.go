// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package k8s

import (
	"context"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var gatewayGVR = schema.GroupVersionResource{
	Group:    "gateway.networking.k8s.io",
	Version:  "v1",
	Resource: "gateways",
}

const gatewayAddressTypeHostname = "Hostname"

// GetGatewayHostname returns the first Hostname address published in the
// status of the named Gateway. Gateway controllers copy the load balancer
// hostname of the Service backing the Gateway into status.addresses, so this
// identifies the exact load balancer for the Gateway regardless of what other
// Services exist in its namespace.
func (kc *KubeClient) GetGatewayHostname(namespace, name string) (string, error) {
	ctx := context.TODO()
	gateway, err := kc.DynamicClient.Resource(gatewayGVR).Namespace(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return "", errors.Wrapf(err, "failed to get gateway %s/%s", namespace, name)
	}

	addresses, _, err := unstructured.NestedSlice(gateway.Object, "status", "addresses")
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse status.addresses of gateway %s/%s", namespace, name)
	}

	for _, entry := range addresses {
		address, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		// Per the Gateway API spec an address without a type is an IPAddress.
		addressType, _ := address["type"].(string)
		value, _ := address["value"].(string)
		if addressType == gatewayAddressTypeHostname && value != "" {
			return value, nil
		}
	}

	return "", errors.Errorf("gateway %s/%s has no Hostname address in its status", namespace, name)
}
