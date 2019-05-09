package k8s

import (
	rbacv1 "k8s.io/api/rbac/v1"
	rbacbetav1 "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (kc *KubeClient) createClusterRoleV1(account *rbacv1.ClusterRole) (metav1.Object, error) {
	client := kc.Clientset.RbacV1().ClusterRoles()

	result, err := client.Create(account)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}

func (kc *KubeClient) createClusterRoleBetaV1(account *rbacbetav1.ClusterRole) (metav1.Object, error) {
	client := kc.Clientset.RbacV1beta1().ClusterRoles()

	result, err := client.Create(account)
	if err != nil {
		return nil, err
	}

	return result.GetObjectMeta(), nil
}
