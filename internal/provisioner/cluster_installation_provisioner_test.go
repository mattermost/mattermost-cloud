// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/require"
)

func TestAddInternalSourceRangesToAnnotations(t *testing.T) {
	t.Run("nil allowed ranges, blank internal ranges", func(t *testing.T) {
		annotations := getIngressAnnotations()
		addInternalSourceRangesToAnnotations(annotations, nil, "")
		require.Equal(t, getIngressAnnotations(), annotations)
	})

	t.Run("nil allowed ranges, internal ranges", func(t *testing.T) {
		annotations := getIngressAnnotations()
		addInternalSourceRangesToAnnotations(annotations, nil, "2.2.2.2/24")
		require.Equal(t, getIngressAnnotations(), annotations)
	})

	t.Run("allowed ranges, blank internal ranges", func(t *testing.T) {
		annotations := getIngressAnnotations()
		allowedRanges := &model.AllowedIPRanges{{CIDRBlock: "1.1.1.1/24"}}
		addInternalSourceRangesToAnnotations(annotations, allowedRanges, "")
		require.Equal(t, "1.1.1.1/24", annotations["nginx.ingress.kubernetes.io/whitelist-source-range"])
		for k, v := range getIngressAnnotations() {
			require.Equal(t, v, annotations[k])
		}
	})

	t.Run("allowed ranges, blank internal ranges", func(t *testing.T) {
		annotations := getIngressAnnotations()
		allowedRanges := &model.AllowedIPRanges{{CIDRBlock: "1.1.1.1/24"}}
		addInternalSourceRangesToAnnotations(annotations, allowedRanges, "2.2.2.2/24")
		require.Equal(t, "1.1.1.1/24,2.2.2.2/24", annotations["nginx.ingress.kubernetes.io/whitelist-source-range"])
		for k, v := range getIngressAnnotations() {
			require.Equal(t, v, annotations[k])
		}
	})

	t.Run("multiple of both ranges", func(t *testing.T) {
		annotations := getIngressAnnotations()
		allowedRanges := &model.AllowedIPRanges{
			{CIDRBlock: "1.1.1.1/24"},
			{CIDRBlock: "1.1.1.2/24"},
		}
		addInternalSourceRangesToAnnotations(annotations, allowedRanges, "2.2.2.2/24,2.2.2.3/24")
		require.Equal(t, "1.1.1.1/24,1.1.1.2/24,2.2.2.2/24,2.2.2.3/24", annotations["nginx.ingress.kubernetes.io/whitelist-source-range"])
		for k, v := range getIngressAnnotations() {
			require.Equal(t, v, annotations[k])
		}
	})
}
