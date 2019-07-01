package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kc *KubeClient) createOrUpdateStatefulSet(namespace string, stset *appsv1.StatefulSet) (metav1.Object, error) {
	_, err := kc.Clientset.AppsV1().StatefulSets(namespace).Get(stset.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.AppsV1().StatefulSets(namespace).Create(stset)
	}

	return kc.Clientset.AppsV1().StatefulSets(namespace).Update(stset)
}
