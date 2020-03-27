package k8s

import (
	"testing"

	"github.com/stretchr/testify/require"
	v1alpha3 "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha3"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestClusterIssuer(t *testing.T) {
	testClient := newTestKubeClient()
	clusterissuer := &v1alpha3.ClusterIssuer{
		ObjectMeta: v1.ObjectMeta{Name: "test-clusterissuer"},
	}

	t.Run("create clusterissuer", func(t *testing.T) {
		result, err := testClient.createOrUpdateClusterIssuer(clusterissuer)
		require.NoError(t, err)
		require.Equal(t, clusterissuer.GetName(), result.GetName())
	})
	t.Run("create duplicate clusterissuer", func(t *testing.T) {
		result, err := testClient.createOrUpdateClusterIssuer(clusterissuer)
		require.NoError(t, err)
		require.Equal(t, clusterissuer.GetName(), result.GetName())
	})
}
