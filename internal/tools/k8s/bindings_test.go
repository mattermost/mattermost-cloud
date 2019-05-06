package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	rbacbetav1 "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRoleBindingsV1(t *testing.T) {
	testClient := newTestKubeClient()
	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "test-binding"},
	}
	namespace := "testing"

	t.Run("create role binding", func(t *testing.T) {
		result, err := testClient.createRoleBindingV1(namespace, roleBinding)
		assert.NoError(t, err)
		assert.Equal(t, roleBinding.GetName(), result.GetName())
	})
	t.Run("create duplicate role binding", func(t *testing.T) {
		_, err := testClient.createRoleBindingV1(namespace, roleBinding)
		assert.Error(t, err)
	})
}

func TestRoleBindingsBetaV1(t *testing.T) {
	testClient := newTestKubeClient()
	roleBinding := &rbacbetav1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "test-binding"},
	}
	namespace := "testing"

	t.Run("create role binding", func(t *testing.T) {
		result, err := testClient.createRoleBindingBetaV1(namespace, roleBinding)
		assert.NoError(t, err)
		assert.Equal(t, roleBinding.GetName(), result.GetName())
	})
	t.Run("create duplicate role binding", func(t *testing.T) {
		_, err := testClient.createRoleBindingBetaV1(namespace, roleBinding)
		assert.Error(t, err)
	})
}

func TestClusterRoleBindingsV1(t *testing.T) {
	testClient := newTestKubeClient()
	roleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "test-binding"},
	}

	t.Run("create role binding", func(t *testing.T) {
		result, err := testClient.createClusterRoleBindingV1(roleBinding)
		assert.NoError(t, err)
		assert.Equal(t, roleBinding.GetName(), result.GetName())
	})
	t.Run("create duplicate role binding", func(t *testing.T) {
		_, err := testClient.createClusterRoleBindingV1(roleBinding)
		assert.Error(t, err)
	})
}

func TestClusterRoleBindingsBetaV1(t *testing.T) {
	testClient := newTestKubeClient()
	roleBinding := &rbacbetav1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "test-binding"},
	}

	t.Run("create role binding", func(t *testing.T) {
		result, err := testClient.createClusterRoleBindingBetaV1(roleBinding)
		assert.NoError(t, err)
		assert.Equal(t, roleBinding.GetName(), result.GetName())
	})
	t.Run("create duplicate role binding", func(t *testing.T) {
		_, err := testClient.createClusterRoleBindingBetaV1(roleBinding)
		assert.Error(t, err)
	})
}
