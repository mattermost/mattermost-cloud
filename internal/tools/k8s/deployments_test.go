package k8s

import (
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	appsbetav1 "k8s.io/api/apps/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDeploymentsV1(t *testing.T) {
	testClient := newTestKubeClient()
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deployment"},
	}
	namespace := "testing"

	t.Run("create deployment", func(t *testing.T) {
		result, err := testClient.createOrUpdateDeploymentV1(namespace, deployment)
		require.NoError(t, err)
		require.Equal(t, deployment.GetName(), result.GetName())
	})
	t.Run("create duplicate deployment", func(t *testing.T) {
		result, err := testClient.createOrUpdateDeploymentV1(namespace, deployment)
		require.NoError(t, err)
		require.Equal(t, deployment.GetName(), result.GetName())
	})
}

func TestDeploymentsBetaV1(t *testing.T) {
	testClient := newTestKubeClient()
	deployment := &appsbetav1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deployment"},
	}
	namespace := "testing"

	t.Run("create deployment", func(t *testing.T) {
		result, err := testClient.createOrUpdateDeploymentBetaV1(namespace, deployment)
		require.NoError(t, err)
		require.Equal(t, deployment.GetName(), result.GetName())
	})
	t.Run("create duplicate deployment", func(t *testing.T) {
		result, err := testClient.createOrUpdateDeploymentBetaV1(namespace, deployment)
		require.NoError(t, err)
		require.Equal(t, deployment.GetName(), result.GetName())
	})
}
