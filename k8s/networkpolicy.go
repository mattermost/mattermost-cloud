// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"context"

	networkingv1 "k8s.io/api/networking/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	allowMMExternal     = "external-mm-allow"
	allowMMExternalBeta = "external-mm-v1beta-allow"
)

func (kc *KubeClient) createOrUpdateNetworkPolicyV1(namespace string, networkPolicy *networkingv1.NetworkPolicy) (metav1.Object, error) {
	ctx := context.TODO()
	_, err := kc.Clientset.NetworkingV1().NetworkPolicies(namespace).Get(ctx, networkPolicy.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.NetworkingV1().NetworkPolicies(namespace).Create(ctx, networkPolicy, metav1.CreateOptions{})
	}

	return kc.Clientset.NetworkingV1().NetworkPolicies(namespace).Update(ctx, networkPolicy, metav1.UpdateOptions{})
}

func (kc *KubeClient) updateLabelsNetworkPolicy(networkPolicy *networkingv1.NetworkPolicy, installationName string) {
	if networkPolicy.GetName() == allowMMExternal {
		networkPolicy.Spec.PodSelector.MatchLabels = map[string]string{
			"v1alpha1.mattermost.com/installation": installationName,
			"app":                                  "mattermost",
		}
		return
	}
	if networkPolicy.GetName() == allowMMExternalBeta {
		networkPolicy.Spec.PodSelector.MatchLabels = map[string]string{
			"installation.mattermost.com/installation": installationName,
			"app": "mattermost",
		}
		return
	}
}
