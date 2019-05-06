package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNamespaces(t *testing.T) {
	testClient := newTestKubeClient()

	namespaceNames := []string{"namespace1, namespace2, namespace3"}
	t.Run("create namespaces", func(t *testing.T) {
		namespaces, err := testClient.CreateNamespaces(namespaceNames)
		assert.NoError(t, err)

		var names []string
		for _, namespace := range namespaces {
			names = append(names, namespace.GetName())
		}
		assert.Equal(t, namespaceNames, names)
	})
	t.Run("get namespaces", func(t *testing.T) {
		namespaces, err := testClient.GetNamespaces(namespaceNames)
		assert.NoError(t, err)

		var names []string
		for _, namespace := range namespaces {
			names = append(names, namespace.GetName())
		}
		assert.Equal(t, namespaceNames, names)
	})
	t.Run("delete namespaces", func(t *testing.T) {
		err := testClient.DeleteNamespaces(namespaceNames)
		assert.NoError(t, err)
	})
}
