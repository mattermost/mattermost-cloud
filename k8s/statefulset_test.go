// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"testing"

	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestStatefuleSets(t *testing.T) {
	testClient := newTestKubeClient()
	statefulSet := &appsv1.StatefulSet{
		ObjectMeta: metav1.ObjectMeta{Name: "test-deployment"},
	}
	namespace := "testing"

	t.Run("create statefulset", func(t *testing.T) {
		result, err := testClient.createOrUpdateStatefulSet(namespace, statefulSet)
		require.NoError(t, err)
		require.Equal(t, statefulSet.GetName(), result.GetName())
	})
	t.Run("create duplicate statefulset", func(t *testing.T) {
		result, err := testClient.createOrUpdateStatefulSet(namespace, statefulSet)
		require.NoError(t, err)
		require.Equal(t, statefulSet.GetName(), result.GetName())
	})
}
