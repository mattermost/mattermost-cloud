package k8s

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigMaps(t *testing.T) {
	testClient := newTestKubeClient()
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deployment"},
	}
	namespace := "testing"

	t.Run("create configmap", func(t *testing.T) {
		result, err := testClient.createOrUpdateConfigMap(namespace, configMap)
		require.NoError(t, err)
		require.Equal(t, configMap.GetName(), result.GetName())
	})
	t.Run("create duplicate configmap", func(t *testing.T) {
		result, err := testClient.createOrUpdateConfigMap(namespace, configMap)
		require.NoError(t, err)
		require.Equal(t, configMap.GetName(), result.GetName())
	})
}
