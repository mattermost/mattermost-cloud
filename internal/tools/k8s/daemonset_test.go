package k8s

import (
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDaemonSetV1(t *testing.T) {
	testClient := newTestKubeClient()
	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test-daemonSet"},
	}
	namespace := "testing"

	t.Run("create daemonSet", func(t *testing.T) {
		result, err := testClient.createOrUpdateDaemonSetV1(namespace, daemonSet)
		require.NoError(t, err)
		require.Equal(t, daemonSet.GetName(), result.GetName())
	})
	t.Run("create duplicate daemonSet", func(t *testing.T) {
		result, err := testClient.createOrUpdateDaemonSetV1(namespace, daemonSet)
		require.NoError(t, err)
		require.Equal(t, daemonSet.GetName(), result.GetName())
	})
}
