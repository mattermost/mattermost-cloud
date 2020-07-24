// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPersistentVolume(t *testing.T) {
	testClient := newTestKubeClient()
	persistentVolume := &corev1.PersistentVolume{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deployment"},
	}

	t.Run("create service account", func(t *testing.T) {
		result, err := testClient.createOrUpdatePersistentVolume(persistentVolume)
		require.NoError(t, err)
		require.Equal(t, persistentVolume.GetName(), result.GetName())
	})
	t.Run("create duplicate service account", func(t *testing.T) {
		result, err := testClient.createOrUpdatePersistentVolume(persistentVolume)
		require.NoError(t, err)
		require.Equal(t, persistentVolume.GetName(), result.GetName())
	})
}

func TestPersistentVolumeClaim(t *testing.T) {
	testClient := newTestKubeClient()
	persistentVolume := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deployment"},
	}
	namespace := "testing"

	t.Run("create service account", func(t *testing.T) {
		result, err := testClient.createOrUpdatePersistentVolumeClaim(namespace, persistentVolume)
		require.NoError(t, err)
		require.Equal(t, persistentVolume.GetName(), result.GetName())
	})
	t.Run("create duplicate service account", func(t *testing.T) {
		result, err := testClient.createOrUpdatePersistentVolumeClaim(namespace, persistentVolume)
		require.NoError(t, err)
		require.Equal(t, persistentVolume.GetName(), result.GetName())
	})
}
