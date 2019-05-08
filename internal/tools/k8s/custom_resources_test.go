package k8s

import (
	"testing"

	mmv1alpha1 "github.com/mattermost/mattermost-cloud/internal/tools/k8s/pkg/apis/mattermost.com/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apixv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCustomResourceDefinition(t *testing.T) {
	testClient := newTestKubeClient()
	customResourceDefinition := &apixv1beta1.CustomResourceDefinition{
		ObjectMeta: metav1.ObjectMeta{Name: "test-crd"},
	}

	t.Run("create custom resource definition", func(t *testing.T) {
		result, err := testClient.createCustomResourceDefinition(customResourceDefinition)
		require.NoError(t, err)
		assert.Equal(t, customResourceDefinition.GetName(), result.GetName())
	})
	t.Run("create duplicate custom resource definition", func(t *testing.T) {
		_, err := testClient.createCustomResourceDefinition(customResourceDefinition)
		assert.Error(t, err)
	})
}

func TestClusterInstallation(t *testing.T) {
	testClient := newTestKubeClient()
	customResource := &mmv1alpha1.ClusterInstallation{
		ObjectMeta: metav1.ObjectMeta{Name: "test-cr"},
	}
	namespace := "testing"

	t.Run("create custom resource", func(t *testing.T) {
		result, err := testClient.createClusterInstallation(namespace, customResource)
		require.NoError(t, err)
		assert.Equal(t, customResource.GetName(), result.GetName())
	})
	t.Run("create duplicate custom resource", func(t *testing.T) {
		_, err := testClient.createClusterInstallation(namespace, customResource)
		assert.Error(t, err)
	})
}
