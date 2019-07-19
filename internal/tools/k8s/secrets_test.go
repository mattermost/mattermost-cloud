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
		result, err := testClient.CreateOrUpdateSecret(namespace, secret)
		require.NoError(t, err)
		require.Equal(t, secret.GetName(), result.GetName())
	})
	t.Run("create duplicate secret", func(t *testing.T) {
		result, err := testClient.CreateOrUpdateSecret(namespace, secret)
		require.NoError(t, err)
		require.Equal(t, secret.GetName(), result.GetName())
	})

	t.Run("get secrets", func(t *testing.T) {
		secrets, err := testClient.GetSecrets(namespace, []string{"test-deployment"})
		require.NoError(t, err)

		var secretNames []string
		for _, secret := range secrets {
			secretNames = append(secretNames, secret.GetName())
		}

		require.Equal(t, secretNames, []string{"test-deployment"})
	})
}
