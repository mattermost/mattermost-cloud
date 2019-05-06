package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	rbacv1 "k8s.io/api/rbac/v1"
	rbacbetav1 "k8s.io/api/rbac/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClusterRolesV1(t *testing.T) {
	testClient := newTestKubeClient()
	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deployment"},
	}

	t.Run("create cluster role", func(t *testing.T) {
		result, err := testClient.createClusterRoleV1(clusterRole)
		assert.NoError(t, err)
		assert.Equal(t, clusterRole.GetName(), result.GetName())
	})
	t.Run("create duplicate cluster role", func(t *testing.T) {
		_, err := testClient.createClusterRoleV1(clusterRole)
		assert.Error(t, err)
	})
}

func TestClusterRolesBetaV1(t *testing.T) {
	testClient := newTestKubeClient()
	clusterRole := &rbacbetav1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deployment"},
	}

	t.Run("create cluster role", func(t *testing.T) {
		result, err := testClient.createClusterRoleBetaV1(clusterRole)
		assert.NoError(t, err)
		assert.Equal(t, clusterRole.GetName(), result.GetName())
	})
	t.Run("create duplicate cluster role", func(t *testing.T) {
		_, err := testClient.createClusterRoleBetaV1(clusterRole)
		assert.Error(t, err)
	})
}
