package k8s

import (
	networkingv1 "k8s.io/api/networking/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	allowMMExternal = "external-mm-allow"
)

func (kc *KubeClient) createOrUpdateNetworkPolicyV1(namespace string, networkPolicy *networkingv1.NetworkPolicy) (metav1.Object, error) {
	_, err := kc.Clientset.NetworkingV1().NetworkPolicies(namespace).Get(networkPolicy.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.NetworkingV1().NetworkPolicies(namespace).Create(networkPolicy)
	}

	return kc.Clientset.NetworkingV1().NetworkPolicies(namespace).Update(networkPolicy)
}

func (kc *KubeClient) updateLabelsNetworkPolicy(networkPolicy *networkingv1.NetworkPolicy, installationName string) {
	if networkPolicy.GetName() != allowMMExternal {
		return
	}

	networkPolicy.Spec.PodSelector.MatchLabels = map[string]string{
		"v1alpha1.mattermost.com/installation": installationName,
		"app":                                  "mattermost",
	}
}
