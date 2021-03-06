// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"testing"

	"github.com/stretchr/testify/require"
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
		result, err := testClient.createOrUpdateClusterRoleV1(clusterRole)
		require.NoError(t, err)
		require.Equal(t, clusterRole.GetName(), result.GetName())
	})
	t.Run("create duplicate cluster role", func(t *testing.T) {
		result, err := testClient.createOrUpdateClusterRoleV1(clusterRole)
		require.NoError(t, err)
		require.Equal(t, clusterRole.GetName(), result.GetName())
	})
}

func TestClusterRolesBetaV1(t *testing.T) {
	testClient := newTestKubeClient()
	clusterRole := &rbacbetav1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deployment"},
	}

	t.Run("create cluster role", func(t *testing.T) {
		result, err := testClient.createOrUpdateClusterRoleBetaV1(clusterRole)
		require.NoError(t, err)
		require.Equal(t, clusterRole.GetName(), result.GetName())
	})
	t.Run("create duplicate cluster role", func(t *testing.T) {
		result, err := testClient.createOrUpdateClusterRoleBetaV1(clusterRole)
		require.NoError(t, err)
		require.Equal(t, clusterRole.GetName(), result.GetName())
	})
}

func TestRolesV1(t *testing.T) {
	testClient := newTestKubeClient()
	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-role",
			Namespace: "test-role-ns",
		},
	}

	t.Run("create role", func(t *testing.T) {
		result, err := testClient.createOrUpdateRoleV1(role)
		require.NoError(t, err)
		require.Equal(t, role.GetName(), result.GetName())
	})
	t.Run("create cluster role", func(t *testing.T) {
		result, err := testClient.createOrUpdateRoleV1(role)
		require.NoError(t, err)
		require.Equal(t, role.GetName(), result.GetName())
	})
}

func TestRolesBetaV1(t *testing.T) {
	testClient := newTestKubeClient()
	role := &rbacbetav1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-role",
			Namespace: "test-role-ns",
		},
	}

	t.Run("create role", func(t *testing.T) {
		result, err := testClient.createOrUpdateRoleBetaV1(role)
		require.NoError(t, err)
		require.Equal(t, role.GetName(), result.GetName())
	})
	t.Run("create duplicate role", func(t *testing.T) {
		result, err := testClient.createOrUpdateRoleBetaV1(role)
		require.NoError(t, err)
		require.Equal(t, role.GetName(), result.GetName())
	})
}
