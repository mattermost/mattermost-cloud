package k8s

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestServices(t *testing.T) {
	testClient := newTestKubeClient()
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deployment"},
	}
	namespace := "testing"

	t.Run("create service", func(t *testing.T) {
		result, err := testClient.createOrUpdateService(namespace, service)
		require.NoError(t, err)
		require.Equal(t, service.GetName(), result.GetName())
	})
	t.Run("create duplicate service", func(t *testing.T) {
		result, err := testClient.createOrUpdateService(namespace, service)
		require.NoError(t, err)
		require.Equal(t, service.GetName(), result.GetName())
	})
}
