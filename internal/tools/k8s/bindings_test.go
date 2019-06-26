package k8s

import (
	"testing"

	"github.com/stretchr/testify/require"
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
		result, err := testClient.createOrUpdateRoleBindingV1(namespace, roleBinding)
		require.NoError(t, err)
		require.Equal(t, roleBinding.GetName(), result.GetName())
	})
	t.Run("create duplicate role binding", func(t *testing.T) {
		result, err := testClient.createOrUpdateRoleBindingV1(namespace, roleBinding)
		require.NoError(t, err)
		require.Equal(t, roleBinding.GetName(), result.GetName())
	})
}

func TestRoleBindingsBetaV1(t *testing.T) {
	testClient := newTestKubeClient()
	roleBinding := &rbacbetav1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "test-binding"},
	}
	namespace := "testing"

	t.Run("create role binding", func(t *testing.T) {
		result, err := testClient.createOrUpdateRoleBindingBetaV1(namespace, roleBinding)
		require.NoError(t, err)
		require.Equal(t, roleBinding.GetName(), result.GetName())
	})
	t.Run("create duplicate role binding", func(t *testing.T) {
		result, err := testClient.createOrUpdateRoleBindingBetaV1(namespace, roleBinding)
		require.NoError(t, err)
		require.Equal(t, roleBinding.GetName(), result.GetName())
	})
}

func TestClusterRoleBindingsV1(t *testing.T) {
	testClient := newTestKubeClient()
	roleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "test-binding"},
	}

	t.Run("create role binding", func(t *testing.T) {
		result, err := testClient.createOrUpdateClusterRoleBindingV1(roleBinding)
		require.NoError(t, err)
		require.Equal(t, roleBinding.GetName(), result.GetName())
	})
	t.Run("create duplicate role binding", func(t *testing.T) {
		result, err := testClient.createOrUpdateClusterRoleBindingV1(roleBinding)
		require.NoError(t, err)
		require.Equal(t, roleBinding.GetName(), result.GetName())
	})
}

func TestClusterRoleBindingsBetaV1(t *testing.T) {
	testClient := newTestKubeClient()
	roleBinding := &rbacbetav1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "test-binding"},
	}

	t.Run("create role binding", func(t *testing.T) {
		result, err := testClient.createOrUpdateClusterRoleBindingBetaV1(roleBinding)
		require.NoError(t, err)
		require.Equal(t, roleBinding.GetName(), result.GetName())
	})
	t.Run("create duplicate role binding", func(t *testing.T) {
		result, err := testClient.createOrUpdateClusterRoleBindingBetaV1(roleBinding)
		require.NoError(t, err)
		require.Equal(t, roleBinding.GetName(), result.GetName())
	})
}
