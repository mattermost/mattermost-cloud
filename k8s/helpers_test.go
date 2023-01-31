// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"context"
	"testing"
	"time"

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
		Status:     corev1.PodStatus{Phase: corev1.PodPending},
	}

	ctx := context.TODO()
	t.Run("don't wait for running", func(t *testing.T) {
		_, err := testClient.Clientset.CoreV1().Pods(namespace).Create(ctx, &pod, metav1.CreateOptions{})
		require.NoError(t, err)
		ctx2, cancel2 := context.WithCancel(context.Background())
		cancel2()
		_, err = testClient.WaitForPodRunning(ctx2, namespace, podName)
		require.Error(t, err)
		err = testClient.Clientset.CoreV1().Pods(namespace).Delete(ctx2, podName, metav1.DeleteOptions{})
		require.NoError(t, err)
	})
	t.Run("create pod", func(t *testing.T) {
		pod.Status.Phase = corev1.PodRunning
		pod, err := testClient.Clientset.CoreV1().Pods(namespace).Create(ctx, &pod, metav1.CreateOptions{})
		require.NoError(t, err)
		assert.Equal(t, podName, pod.GetName())
		assert.Equal(t, corev1.PodRunning, pod.Status.Phase)
	})
	t.Run("wait for running", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		pod, err := testClient.WaitForPodRunning(ctx, namespace, podName)
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
		result, err := testClient.createOrUpdateDeploymentV1(namespace, deployment)
		require.NoError(t, err)
		assert.Equal(t, deployment.GetName(), result.GetName())
	})
	t.Run("get pods from deployment", func(t *testing.T) {
		pods, err := testClient.GetPodsFromDeployment(namespace, deployment.GetName())
		require.NoError(t, err)
		assert.Len(t, pods.Items, 0)
	})
}

func TestGetPodsFromStatefulset(t *testing.T) {
	testClient := newTestKubeClient()
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test-statefulSet"},
	}
	namespace := "testing"

	t.Run("create statefullSet", func(t *testing.T) {
		result, err := testClient.createOrUpdateStatefulSet(namespace, statefulSet)
		require.NoError(t, err)
		assert.Equal(t, statefulSet.GetName(), result.GetName())
	})
	t.Run("get pods from statefullSet", func(t *testing.T) {
		pods, err := testClient.GetPodsFromStatefulset(namespace, statefulSet.GetName())
		require.NoError(t, err)
		assert.Len(t, pods.Items, 0)
	})
}

func TestGetPodsFromDaemonSet(t *testing.T) {
	testClient := newTestKubeClient()
	daemonSet := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test-daemonSet"},
	}
	namespace := "testing"

	t.Run("create daemonSet", func(t *testing.T) {
		result, err := testClient.createOrUpdateDaemonSetV1(namespace, daemonSet)
		require.NoError(t, err)
		assert.Equal(t, daemonSet.GetName(), result.GetName())
	})
	t.Run("get pods from daemonSet", func(t *testing.T) {
		pods, err := testClient.GetPodsFromDaemonSet(namespace, daemonSet.GetName())
		require.NoError(t, err)
		assert.Len(t, pods.Items, 0)
	})
}
