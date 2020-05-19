package k8s

import (
	"testing"

	"github.com/stretchr/testify/require"
	v1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPodDisruptionBudgetBetaV1(t *testing.T) {
	testClient := newTestKubeClient()
	podDisruptionBudget := &v1beta1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{Name: "test-podDisruptionBudget"},
	}
	namespace := "testing"

	t.Run("create PodDisruptionBudget", func(t *testing.T) {
		result, err := testClient.createOrUpdatePodDisruptionBudgetBetaV1(namespace, podDisruptionBudget)
		require.NoError(t, err)
		require.Equal(t, podDisruptionBudget.GetName(), result.GetName())
	})
	t.Run("create duplicate PodDisruptionBudget", func(t *testing.T) {
		result, err := testClient.createOrUpdatePodDisruptionBudgetBetaV1(namespace, podDisruptionBudget)
		require.NoError(t, err)
		require.Equal(t, podDisruptionBudget.GetName(), result.GetName())
	})
}
