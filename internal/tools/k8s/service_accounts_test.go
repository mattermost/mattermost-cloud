package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestServiceAccounts(t *testing.T) {
	testClient := newTestKubeClient()
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deployment"},
	}
	namespace := "testing"

	t.Run("create service account", func(t *testing.T) {
		result, err := testClient.createServiceAccount(namespace, serviceAccount)
		require.NoError(t, err)
		assert.Equal(t, serviceAccount.GetName(), result.GetName())
	})
	t.Run("create duplicate service account", func(t *testing.T) {
		_, err := testClient.createServiceAccount(namespace, serviceAccount)
		assert.Error(t, err)
	})
}
