// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNamespaces(t *testing.T) {
	testClient := newTestKubeClient()

	namespaceNames := []string{"namespace1, namespace2, namespace3"}
	t.Run("create namespaces", func(t *testing.T) {
		namespaces, err := testClient.CreateOrUpdateNamespaces(namespaceNames)
		require.NoError(t, err)

		var names []string
		var nameSpaceLabel []string
		for _, namespace := range namespaces {
			names = append(names, namespace.GetName())
			nameSpaceLabel = append(nameSpaceLabel, namespace.GetLabels()["name"])
		}
		assert.Equal(t, namespaceNames, names)
		assert.Equal(t, namespaceNames, nameSpaceLabel)
	})
	t.Run("get namespaces", func(t *testing.T) {
		namespaces, err := testClient.GetNamespaces(namespaceNames)
		require.NoError(t, err)

		var names []string
		for _, namespace := range namespaces {
			names = append(names, namespace.GetName())
		}
		assert.Equal(t, namespaceNames, names)
	})
	t.Run("update namespace", func(t *testing.T) {
		testNamespace := "update-namespace"
		nsSpec := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}
		_, err := testClient.Clientset.CoreV1().Namespaces().Create(nsSpec)
		require.NoError(t, err)

		namespace, err := testClient.CreateOrUpdateNamespaces([]string{testNamespace})
		require.NoError(t, err)

		assert.Len(t, namespace, 1)
		assert.Equal(t, testNamespace, namespace[0].GetName())
		assert.Equal(t, testNamespace, namespace[0].GetLabels()["name"])

		err = testClient.DeleteNamespaces([]string{testNamespace})
		require.NoError(t, err)

	})
	t.Run("delete namespaces", func(t *testing.T) {
		err := testClient.DeleteNamespaces(namespaceNames)
		require.NoError(t, err)
	})
}
