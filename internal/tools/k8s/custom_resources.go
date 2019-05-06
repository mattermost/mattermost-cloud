package k8s

import (
	mmv1alpha1 "github.com/mattermost/mattermost-cloud/internal/tools/k8s/pkg/apis/mattermost.com/v1alpha1"
	apixv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kc *KubeClient) createCustomResourceDefinition(crd *apixv1beta1.CustomResourceDefinition) (metav1.Object, error) {
	client := kc.ApixClientset.ApiextensionsV1beta1().CustomResourceDefinitions()

	result, err := client.Create(crd)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}

func (kc *KubeClient) createClusterInstallation(namespace string, ci *mmv1alpha1.ClusterInstallation) (metav1.Object, error) {
	client := kc.MattermostClientset.ExampleV1alpha1().ClusterInstallations(namespace)

	result, err := client.Create(ci)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}
