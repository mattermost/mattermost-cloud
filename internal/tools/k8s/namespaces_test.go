package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNamespaces(t *testing.T) {
	testClient := newTestKubeClient()

	namespaceNames := []string{"namespace1, namespace2, namespace3"}
	t.Run("create namespaces", func(t *testing.T) {
		namespaces, err := testClient.CreateNamespaces(namespaceNames)
		require.NoError(t, err)

		var names []string
		for _, namespace := range namespaces {
			names = append(names, namespace.GetName())
		}
		assert.Equal(t, namespaceNames, names)
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
	t.Run("delete namespaces", func(t *testing.T) {
		err := testClient.DeleteNamespaces(namespaceNames)
		require.NoError(t, err)
	})
}
