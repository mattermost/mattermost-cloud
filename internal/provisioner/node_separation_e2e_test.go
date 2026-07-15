// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/mattermost/mattermost-cloud/model"
	mmv1beta1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8syaml "sigs.k8s.io/yaml"
)

// TestNodeSeparation_EndToEnd exercises the full SEC-9253 opt-in path exactly as
// the provisioner does on reconcile:
//
//	installation annotation (in the store)  ->  resolveNodeSeparation
//	                                        ->  ensureScheduling (single owner)
//	                                        ->  the Mattermost CR the operator applies
//
// It renders the resulting CR scheduling spec (ResourceLabels + Affinity) as YAML
// so a reviewer can see what actually lands on the pods for an opted-in
// (community/hub) installation versus an ordinary one.
func TestNodeSeparation_EndToEnd(t *testing.T) {
	logger := testlib.MakeLogger(t)
	sqlStore := store.MakeTestSQLStore(t, logger)
	defer store.CloseConnection(t, sqlStore)

	// An opted-in installation (e.g. `community` or `hub`) carries the
	// `separate-nodes` annotation alongside unrelated ones.
	optedIn := &model.Installation{Name: "community"}
	err := sqlStore.CreateInstallation(optedIn, []*model.Annotation{
		{Name: "internal"},
		{Name: nodeSeparationAnnotation},
	}, nil)
	require.NoError(t, err)

	// A regular customer installation without the annotation.
	optedOut := &model.Installation{Name: "customer"}
	err = sqlStore.CreateInstallation(optedOut, []*model.Annotation{
		{Name: "multi-tenant"},
	}, nil)
	require.NoError(t, err)

	provisioner := Provisioner{store: sqlStore}

	renderScheduling := func(t *testing.T, installation *model.Installation) (bool, mmv1beta1.MattermostSpec) {
		t.Helper()

		// This is the exact store lookup the provisioner performs at reconcile.
		nodeSeparation, err := provisioner.resolveNodeSeparation(installation)
		require.NoError(t, err)

		clusterInstallation := &model.ClusterInstallation{
			ID:        installation.ID + "-ci",
			Namespace: installation.ID + "-ns",
		}
		cluster := &model.Cluster{Name: "internal-cluster"}

		mattermost := &mmv1beta1.Mattermost{
			ObjectMeta: metav1.ObjectMeta{Name: installation.Name},
		}
		// ensureScheduling is the single owner of ResourceLabels + Affinity.
		ensureScheduling(mattermost, installation, clusterInstallation, cluster, nodeSeparation)

		// Only render the scheduling-relevant slice of the CR spec.
		rendered := mmv1beta1.MattermostSpec{
			ResourceLabels: mattermost.Spec.ResourceLabels,
			Scheduling:     mattermost.Spec.Scheduling,
		}
		return nodeSeparation, rendered
	}

	t.Run("opted-in installation gets cross-installation node separation", func(t *testing.T) {
		nodeSeparation, spec := renderScheduling(t, optedIn)
		require.True(t, nodeSeparation, "installation with the separate-nodes annotation must opt in")

		// The separation label is stamped on the pods (via ResourceLabels).
		assert.Equal(t, nodeSeparationAnnotation, spec.ResourceLabels[nodeSeparationLabel],
			"opted-in pods must carry the node-separation-group label")

		terms := spec.Scheduling.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution
		require.Len(t, terms, 3, "two self-spread terms plus the cross-installation separation term")

		sep := terms[2]
		assert.Equal(t, int32(100), sep.Weight)
		assert.Equal(t, "kubernetes.io/hostname", sep.PodAffinityTerm.TopologyKey,
			"separation must be at the node level")
		// Cross-installation: matches the separation label in ALL namespaces.
		require.NotNil(t, sep.PodAffinityTerm.NamespaceSelector)
		assert.Equal(t, &metav1.LabelSelector{}, sep.PodAffinityTerm.NamespaceSelector)
		assert.Empty(t, sep.PodAffinityTerm.Namespaces)
		assert.Equal(t, map[string]string{nodeSeparationLabel: nodeSeparationAnnotation},
			sep.PodAffinityTerm.LabelSelector.MatchLabels)

		yamlOut, err := k8syaml.Marshal(spec)
		require.NoError(t, err)
		t.Logf("\n===== OPTED-IN (community) rendered Mattermost CR scheduling spec =====\n%s", string(yamlOut))
	})

	t.Run("regular installation is unchanged", func(t *testing.T) {
		nodeSeparation, spec := renderScheduling(t, optedOut)
		require.False(t, nodeSeparation, "installation without the annotation must not opt in")

		assert.NotContains(t, spec.ResourceLabels, nodeSeparationLabel,
			"non-opted-in pods must not carry the separation label")

		terms := spec.Scheduling.Affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution
		require.Len(t, terms, 2, "only the two self-spread terms, no cross-installation term")
		for _, term := range terms {
			assert.Nil(t, term.PodAffinityTerm.NamespaceSelector,
				"self-spread terms stay scoped to the installation's own namespace")
		}

		yamlOut, err := k8syaml.Marshal(spec)
		require.NoError(t, err)
		t.Logf("\n===== OPTED-OUT (customer) rendered Mattermost CR scheduling spec =====\n%s", string(yamlOut))
	})
}
