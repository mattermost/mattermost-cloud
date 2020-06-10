package k8s

import (
	v1beta1 "k8s.io/api/policy/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kc *KubeClient) createOrUpdatePodDisruptionBudgetBetaV1(namespace string, podDisruptionBudget *v1beta1.PodDisruptionBudget) (metav1.Object, error) {
	_, err := kc.Clientset.PolicyV1beta1().PodDisruptionBudgets(namespace).Get(podDisruptionBudget.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.PolicyV1beta1().PodDisruptionBudgets(namespace).Create(podDisruptionBudget)
	}

	return kc.Clientset.PolicyV1beta1().PodDisruptionBudgets(namespace).Update(podDisruptionBudget)
}
