// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddSourceRangeWhitelistToAnnotations(t *testing.T) {
	t.Run("nil allowed ranges, blank internal ranges", func(t *testing.T) {
		annotations := getIngressAnnotations()
		addSourceRangeWhitelistToAnnotations(annotations, nil, []string{""})
		require.Equal(t, getIngressAnnotations(), annotations)
	})

	t.Run("nil allowed ranges, internal ranges", func(t *testing.T) {
		annotations := getIngressAnnotations()
		addSourceRangeWhitelistToAnnotations(annotations, nil, []string{"2.2.2.2/24"})
		require.Equal(t, getIngressAnnotations(), annotations)
	})

	t.Run("allowed ranges, blank internal ranges", func(t *testing.T) {
		annotations := getIngressAnnotations()
		allowedRanges := &model.AllowedIPRanges{{CIDRBlock: "1.1.1.1/24", Enabled: true}}
		addSourceRangeWhitelistToAnnotations(annotations, allowedRanges, nil)
		require.Equal(t, []string{"1.1.1.1/24"}, annotations.WhitelistSourceRange)
		expectedAnnotations := getIngressAnnotations()
		expectedAnnotations.WhitelistSourceRange = []string{"1.1.1.1/24"}
		require.Equal(t, annotations, expectedAnnotations)
	})

	t.Run("allowed range, internal range", func(t *testing.T) {
		annotations := getIngressAnnotations()
		allowedRanges := &model.AllowedIPRanges{{CIDRBlock: "1.1.1.1/24", Enabled: true}}
		addSourceRangeWhitelistToAnnotations(annotations, allowedRanges, []string{"2.2.2.2/24"})
		require.Equal(t, []string{"1.1.1.1/24", "2.2.2.2/24"}, annotations.WhitelistSourceRange)
		expectedAnnotations := getIngressAnnotations()
		expectedAnnotations.WhitelistSourceRange = []string{"1.1.1.1/24", "2.2.2.2/24"}
		require.Equal(t, annotations, expectedAnnotations)
	})

	t.Run("multiple of both ranges", func(t *testing.T) {
		annotations := getIngressAnnotations()
		allowedRanges := &model.AllowedIPRanges{
			{CIDRBlock: "1.1.1.1/24", Enabled: true},
			{CIDRBlock: "1.1.1.2/24", Enabled: true},
		}
		addSourceRangeWhitelistToAnnotations(annotations, allowedRanges, []string{"2.2.2.2/24", "2.2.2.3/24"})
		require.Equal(t, []string{"1.1.1.1/24", "1.1.1.2/24", "2.2.2.2/24", "2.2.2.3/24"}, annotations.WhitelistSourceRange)
		expectedAnnotations := getIngressAnnotations()
		expectedAnnotations.WhitelistSourceRange = []string{"1.1.1.1/24", "1.1.1.2/24", "2.2.2.2/24", "2.2.2.3/24"}
		require.Equal(t, annotations, expectedAnnotations)
	})

	t.Run("multiple of both ranges, some disabled allowed ranges", func(t *testing.T) {
		annotations := getIngressAnnotations()
		allowedRanges := &model.AllowedIPRanges{
			{CIDRBlock: "1.1.1.1/24", Enabled: true},
			{CIDRBlock: "1.1.1.2/24", Enabled: false},
		}
		addSourceRangeWhitelistToAnnotations(annotations, allowedRanges, []string{"2.2.2.2/24", "2.2.2.3/24"})
		require.Equal(t, []string{"1.1.1.1/24", "2.2.2.2/24", "2.2.2.3/24"}, annotations.WhitelistSourceRange)
		expectedAnnotations := getIngressAnnotations()
		expectedAnnotations.WhitelistSourceRange = []string{"1.1.1.1/24", "2.2.2.2/24", "2.2.2.3/24"}
		require.Equal(t, annotations, expectedAnnotations)
	})
}

func TestClusterInstallationBaseLabels(t *testing.T) {
	testCases := []struct {
		name                string
		installation        *model.Installation
		clusterInstallation *model.ClusterInstallation
		cluster             *model.Cluster
		expected            map[string]string
	}{
		{
			name: "with cluster name",
			installation: &model.Installation{
				ID: "test-installation",
			},
			clusterInstallation: &model.ClusterInstallation{
				ID: "test-cluster-installation",
			},
			cluster: &model.Cluster{
				Name: "test-cluster",
			},
			expected: map[string]string{
				"installation-id":         "test-installation",
				"cluster-installation-id": "test-cluster-installation",
				"dns":                     "test-cluster-public",
			},
		},
		{
			name: "with empty cluster name",
			installation: &model.Installation{
				ID: "test-installation",
			},
			clusterInstallation: &model.ClusterInstallation{
				ID: "test-cluster-installation",
			},
			cluster: &model.Cluster{
				Name: "",
			},
			expected: map[string]string{
				"installation-id":         "test-installation",
				"cluster-installation-id": "test-cluster-installation",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			labels := clusterInstallationBaseLabels(tc.installation, tc.clusterInstallation, tc.cluster)
			assert.Equal(t, tc.expected, labels)
		})
	}
}
