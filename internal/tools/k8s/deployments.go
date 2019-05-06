package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	appsbetav1 "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kc *KubeClient) createDeploymentV1(namespace string, deployment *appsv1.Deployment) (metav1.Object, error) {
	client := kc.Clientset.AppsV1().Deployments(namespace)

	result, err := client.Create(deployment)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}

func (kc *KubeClient) createDeploymentBetaV1(namespace string, deployment *appsbetav1.Deployment) (metav1.Object, error) {
	client := kc.Clientset.AppsV1beta1().Deployments(namespace)

	result, err := client.Create(deployment)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}
