package k8s

import (
	rbacv1 "k8s.io/api/rbac/v1"
	rbacbetav1 "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kc *KubeClient) createRoleBindingV1(namespace string, binding *rbacv1.RoleBinding) (metav1.Object, error) {
	client := kc.Clientset.RbacV1().RoleBindings(namespace)

	result, err := client.Create(binding)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}

func (kc *KubeClient) createRoleBindingBetaV1(namespace string, binding *rbacbetav1.RoleBinding) (metav1.Object, error) {
	client := kc.Clientset.RbacV1beta1().RoleBindings(namespace)

	result, err := client.Create(binding)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}

func (kc *KubeClient) createClusterRoleBindingV1(binding *rbacv1.ClusterRoleBinding) (metav1.Object, error) {
	client := kc.Clientset.RbacV1().ClusterRoleBindings()

	result, err := client.Create(binding)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}

func (kc *KubeClient) createClusterRoleBindingBetaV1(binding *rbacbetav1.ClusterRoleBinding) (metav1.Object, error) {
	client := kc.Clientset.RbacV1beta1().ClusterRoleBindings()

	result, err := client.Create(binding)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}
