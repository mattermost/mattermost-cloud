package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestWaitForPodRunning(t *testing.T) {
	testClient := newTestKubeClient()
	namespace := "testing"
	podName := "test-pod"
	pod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: podName},
		Status:     corev1.PodStatus{Phase: corev1.PodRunning},
	}

	t.Run("create pod", func(t *testing.T) {
		pod, err := testClient.Clientset.CoreV1().Pods(namespace).Create(&pod)
		require.NoError(t, err)
		assert.Equal(t, podName, pod.GetName())
		assert.Equal(t, corev1.PodRunning, pod.Status.Phase)
	})
	t.Run("wait for running", func(t *testing.T) {
		pod, err := testClient.WaitForPodRunning(namespace, podName, 30)
		require.NoError(t, err)
		assert.Equal(t, corev1.PodRunning, pod.Status.Phase)
	})
}

func TestGetPodsFromDeployment(t *testing.T) {
	testClient := newTestKubeClient()
	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deployment"},
	}
	namespace := "testing"

	t.Run("create deployment", func(t *testing.T) {
		result, err := testClient.createDeploymentV1(namespace, deployment)
		require.NoError(t, err)
		assert.Equal(t, deployment.GetName(), result.GetName())
	})
	t.Run("get pods from deployment", func(t *testing.T) {
		pods, err := testClient.GetPodsFromDeployment(namespace, deployment.GetName())
		require.NoError(t, err)
		assert.Len(t, pods.Items, 0)
	})
}
