package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
		result, err := testClient.createDeploymentV1(namespace, deployment)
		assert.NoError(t, err)
		assert.Equal(t, deployment.GetName(), result.GetName())
	})
	t.Run("create duplicate deployment", func(t *testing.T) {
		_, err := testClient.createDeploymentV1(namespace, deployment)
		assert.Error(t, err)
	})
}

func TestDeploymentsBetaV1(t *testing.T) {
	testClient := newTestKubeClient()
	deployment := &appsbetav1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deployment"},
	}
	namespace := "testing"

	t.Run("create deployment", func(t *testing.T) {
		result, err := testClient.createDeploymentBetaV1(namespace, deployment)
		assert.NoError(t, err)
		assert.Equal(t, deployment.GetName(), result.GetName())
	})
	t.Run("create duplicate deployment", func(t *testing.T) {
		_, err := testClient.createDeploymentBetaV1(namespace, deployment)
		assert.Error(t, err)
	})
}
