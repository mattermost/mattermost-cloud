package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	appsbetav1 "k8s.io/api/apps/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kc *KubeClient) createOrUpdateDeploymentV1(namespace string, deployment *appsv1.Deployment) (metav1.Object, error) {
	_, err := kc.Clientset.AppsV1().Deployments(namespace).Get(deployment.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.AppsV1().Deployments(namespace).Create(deployment)
	}

	return kc.Clientset.AppsV1().Deployments(namespace).Update(deployment)
}

func (kc *KubeClient) createOrUpdateDeploymentBetaV1(namespace string, deployment *appsbetav1.Deployment) (metav1.Object, error) {
	_, err := kc.Clientset.AppsV1beta1().Deployments(namespace).Get(deployment.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.Clientset.AppsV1beta1().Deployments(namespace).Create(deployment)
	}

	return kc.Clientset.AppsV1beta1().Deployments(namespace).Update(deployment)
}
