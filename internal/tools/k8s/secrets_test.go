package k8s

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestSecrets(t *testing.T) {
	testClient := newTestKubeClient()
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deployment"},
	}
	namespace := "testing"

	t.Run("create secret", func(t *testing.T) {
		result, err := testClient.createOrUpdateSecret(namespace, secret)
		require.NoError(t, err)
		require.Equal(t, secret.GetName(), result.GetName())
	})
	t.Run("create duplicate secret", func(t *testing.T) {
		result, err := testClient.createOrUpdateSecret(namespace, secret)
		require.NoError(t, err)
		require.Equal(t, secret.GetName(), result.GetName())
	})
}
