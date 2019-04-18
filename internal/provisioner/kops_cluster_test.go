package provisioner

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestKopsCluster(t *testing.T) {
	kopsCluster := NewKopsCluster("aws")
	require.NotEmpty(t, kopsCluster.ID)
	require.Equal(t, "aws", kopsCluster.Provider)
	require.Equal(t, "kops", kopsCluster.Provisioner)
	require.Equal(t, fmt.Sprintf("%s-kops.k8s.local", kopsCluster.ID), kopsCluster.KopsName())
	require.Equal(t, fmt.Sprintf("%s-kops.k8s.local", kopsCluster.ID), kopsCluster.GetKopsMetadata().Name)

	rebuiltKopsCluster := KopsClusterFromCluster(&kopsCluster.Cluster)
	require.Equal(t, kopsCluster, rebuiltKopsCluster)

	changedKopsCluster := NewKopsCluster("aws")
	changedKopsCluster.SetKopsMetadata(KopsMetadata{Name: "changed"})
	require.Equal(t, "changed", changedKopsCluster.KopsName())

	rebuiltChangedKopsCluster := KopsClusterFromCluster(&changedKopsCluster.Cluster)
	require.Equal(t, changedKopsCluster, rebuiltChangedKopsCluster)
}
