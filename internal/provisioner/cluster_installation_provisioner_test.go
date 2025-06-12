// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	mmv1beta1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1beta1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func TestEnsurePodProbeOverrides(t *testing.T) {
	t.Run("no probe overrides", func(t *testing.T) {
		provisioner := Provisioner{
			params: ProvisioningParams{
				PodProbeOverrides: model.PodProbeOverrides{},
			},
		}

		mattermost := &mmv1beta1.Mattermost{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-mattermost",
			},
			Spec: mmv1beta1.MattermostSpec{
				// Set some initial probe values to ensure they get cleared
				Probes: mmv1beta1.Probes{
					LivenessProbe: corev1.Probe{
						FailureThreshold: 5,
						TimeoutSeconds:   10,
					},
					ReadinessProbe: corev1.Probe{
						FailureThreshold: 3,
						TimeoutSeconds:   5,
					},
				},
			},
		}

		provisioner.ensurePodProbeOverrides(mattermost, nil)

		// Both probes should be cleared to empty Probe structs
		assert.Equal(t, corev1.Probe{}, mattermost.Spec.Probes.LivenessProbe)
		assert.Equal(t, corev1.Probe{}, mattermost.Spec.Probes.ReadinessProbe)
	})

	t.Run("only liveness probe override", func(t *testing.T) {
		livenessOverride := &corev1.Probe{
			FailureThreshold:    10,
			SuccessThreshold:    1,
			InitialDelaySeconds: 60,
			PeriodSeconds:       30,
			TimeoutSeconds:      15,
		}

		provisioner := Provisioner{
			params: ProvisioningParams{
				PodProbeOverrides: PodProbeOverrides{
					LivenessProbeOverride:  livenessOverride,
					ReadinessProbeOverride: nil,
				},
			},
		}

		mattermost := &mmv1beta1.Mattermost{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-mattermost",
			},
			Spec: mmv1beta1.MattermostSpec{
				Probes: mmv1beta1.Probes{
					ReadinessProbe: corev1.Probe{
						FailureThreshold: 3,
						TimeoutSeconds:   5,
					},
				},
			},
		}

		provisioner.ensurePodProbeOverrides(mattermost, nil)

		// Liveness probe should be set to the override
		assert.Equal(t, *livenessOverride, mattermost.Spec.Probes.LivenessProbe)
		// Readiness probe should be cleared
		assert.Equal(t, corev1.Probe{}, mattermost.Spec.Probes.ReadinessProbe)
	})

	t.Run("only readiness probe override", func(t *testing.T) {
		readinessOverride := &corev1.Probe{
			FailureThreshold:    5,
			SuccessThreshold:    2,
			InitialDelaySeconds: 45,
			PeriodSeconds:       20,
			TimeoutSeconds:      10,
		}

		provisioner := Provisioner{
			params: ProvisioningParams{
				PodProbeOverrides: PodProbeOverrides{
					LivenessProbeOverride:  nil,
					ReadinessProbeOverride: readinessOverride,
				},
			},
		}

		mattermost := &mmv1beta1.Mattermost{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-mattermost",
			},
			Spec: mmv1beta1.MattermostSpec{
				Probes: mmv1beta1.Probes{
					LivenessProbe: corev1.Probe{
						FailureThreshold: 8,
						TimeoutSeconds:   12,
					},
				},
			},
		}

		provisioner.ensurePodProbeOverrides(mattermost, nil)

		// Liveness probe should be cleared
		assert.Equal(t, corev1.Probe{}, mattermost.Spec.Probes.LivenessProbe)
		// Readiness probe should be set to the override
		assert.Equal(t, *readinessOverride, mattermost.Spec.Probes.ReadinessProbe)
	})

	t.Run("both probe overrides", func(t *testing.T) {
		livenessOverride := &corev1.Probe{
			FailureThreshold:    12,
			SuccessThreshold:    1,
			InitialDelaySeconds: 90,
			PeriodSeconds:       25,
			TimeoutSeconds:      20,
		}

		readinessOverride := &corev1.Probe{
			FailureThreshold:    8,
			SuccessThreshold:    3,
			InitialDelaySeconds: 30,
			PeriodSeconds:       15,
			TimeoutSeconds:      8,
		}

		provisioner := Provisioner{
			params: ProvisioningParams{
				PodProbeOverrides: PodProbeOverrides{
					LivenessProbeOverride:  livenessOverride,
					ReadinessProbeOverride: readinessOverride,
				},
			},
		}

		mattermost := &mmv1beta1.Mattermost{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-mattermost",
			},
			Spec: mmv1beta1.MattermostSpec{},
		}

		provisioner.ensurePodProbeOverrides(mattermost, nil)

		// Both probes should be set to their respective overrides
		assert.Equal(t, *livenessOverride, mattermost.Spec.Probes.LivenessProbe)
		assert.Equal(t, *readinessOverride, mattermost.Spec.Probes.ReadinessProbe)
	})

	t.Run("overrides replace existing values", func(t *testing.T) {
		livenessOverride := &corev1.Probe{
			FailureThreshold: 7,
			TimeoutSeconds:   25,
		}

		provisioner := Provisioner{
			params: ProvisioningParams{
				PodProbeOverrides: PodProbeOverrides{
					LivenessProbeOverride:  livenessOverride,
					ReadinessProbeOverride: nil,
				},
			},
		}

		mattermost := &mmv1beta1.Mattermost{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-mattermost",
			},
			Spec: mmv1beta1.MattermostSpec{
				Probes: mmv1beta1.Probes{
					LivenessProbe: corev1.Probe{
						FailureThreshold:    999,
						SuccessThreshold:    999,
						InitialDelaySeconds: 999,
						PeriodSeconds:       999,
						TimeoutSeconds:      999,
					},
					ReadinessProbe: corev1.Probe{
						FailureThreshold:    888,
						SuccessThreshold:    888,
						InitialDelaySeconds: 888,
						PeriodSeconds:       888,
						TimeoutSeconds:      888,
					},
				},
			},
		}

		provisioner.ensurePodProbeOverrides(mattermost, nil)

		// Liveness probe should be completely replaced with the override
		assert.Equal(t, *livenessOverride, mattermost.Spec.Probes.LivenessProbe)
		// Readiness probe should be cleared (not the old values)
		assert.Equal(t, corev1.Probe{}, mattermost.Spec.Probes.ReadinessProbe)
	})

	t.Run("partial probe configuration", func(t *testing.T) {
		// Test that we can override with partial probe configuration
		livenessOverride := &corev1.Probe{
			FailureThreshold: 15,
			// Only setting one field, others should be zero values
		}

		readinessOverride := &corev1.Probe{
			InitialDelaySeconds: 120,
			TimeoutSeconds:      30,
			// Only setting two fields, others should be zero values
		}

		provisioner := Provisioner{
			params: ProvisioningParams{
				PodProbeOverrides: PodProbeOverrides{
					LivenessProbeOverride:  livenessOverride,
					ReadinessProbeOverride: readinessOverride,
				},
			},
		}

		mattermost := &mmv1beta1.Mattermost{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-mattermost",
			},
			Spec: mmv1beta1.MattermostSpec{},
		}

		provisioner.ensurePodProbeOverrides(mattermost, nil)

		expectedLiveness := corev1.Probe{
			FailureThreshold: 15,
			// All other fields should be zero values
		}
		assert.Equal(t, expectedLiveness, mattermost.Spec.Probes.LivenessProbe)

		expectedReadiness := corev1.Probe{
			InitialDelaySeconds: 120,
			TimeoutSeconds:      30,
			// All other fields should be zero values
		}
		assert.Equal(t, expectedReadiness, mattermost.Spec.Probes.ReadinessProbe)
	})
}

func TestEnsurePodProbeOverrides_InstallationLevel(t *testing.T) {
	t.Run("installation overrides with no server overrides", func(t *testing.T) {
		livenessOverride := &corev1.Probe{
			FailureThreshold:    8,
			SuccessThreshold:    1,
			InitialDelaySeconds: 45,
			PeriodSeconds:       20,
			TimeoutSeconds:      12,
		}

		readinessOverride := &corev1.Probe{
			FailureThreshold:    6,
			SuccessThreshold:    2,
			InitialDelaySeconds: 35,
			PeriodSeconds:       15,
			TimeoutSeconds:      8,
		}

		provisioner := Provisioner{
			params: ProvisioningParams{
				PodProbeOverrides: PodProbeOverrides{}, // No server overrides
			},
		}

		installation := &model.Installation{
			PodProbeOverrides: &model.PodProbeOverrides{
				LivenessProbeOverride:  livenessOverride,
				ReadinessProbeOverride: readinessOverride,
			},
		}

		mattermost := &mmv1beta1.Mattermost{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-mattermost",
			},
			Spec: mmv1beta1.MattermostSpec{},
		}

		provisioner.ensurePodProbeOverrides(mattermost, installation)

		// Both probes should be set to installation overrides
		assert.Equal(t, *livenessOverride, mattermost.Spec.Probes.LivenessProbe)
		assert.Equal(t, *readinessOverride, mattermost.Spec.Probes.ReadinessProbe)
	})

	t.Run("installation overrides override server overrides", func(t *testing.T) {
		serverLivenessOverride := &corev1.Probe{
			FailureThreshold:    5,
			SuccessThreshold:    1,
			InitialDelaySeconds: 30,
			PeriodSeconds:       10,
			TimeoutSeconds:      5,
		}

		serverReadinessOverride := &corev1.Probe{
			FailureThreshold:    3,
			SuccessThreshold:    1,
			InitialDelaySeconds: 20,
			PeriodSeconds:       8,
			TimeoutSeconds:      3,
		}

		installationLivenessOverride := &corev1.Probe{
			FailureThreshold:    12,
			SuccessThreshold:    2,
			InitialDelaySeconds: 60,
			PeriodSeconds:       25,
			TimeoutSeconds:      15,
		}

		installationReadinessOverride := &corev1.Probe{
			FailureThreshold:    10,
			SuccessThreshold:    3,
			InitialDelaySeconds: 50,
			PeriodSeconds:       20,
			TimeoutSeconds:      12,
		}

		provisioner := Provisioner{
			params: ProvisioningParams{
				PodProbeOverrides: PodProbeOverrides{
					LivenessProbeOverride:  serverLivenessOverride,
					ReadinessProbeOverride: serverReadinessOverride,
				},
			},
		}

		installation := &model.Installation{
			PodProbeOverrides: &model.PodProbeOverrides{
				LivenessProbeOverride:  installationLivenessOverride,
				ReadinessProbeOverride: installationReadinessOverride,
			},
		}

		mattermost := &mmv1beta1.Mattermost{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-mattermost",
			},
			Spec: mmv1beta1.MattermostSpec{},
		}

		provisioner.ensurePodProbeOverrides(mattermost, installation)

		// Both probes should be set to installation overrides (not server overrides)
		assert.Equal(t, *installationLivenessOverride, mattermost.Spec.Probes.LivenessProbe)
		assert.Equal(t, *installationReadinessOverride, mattermost.Spec.Probes.ReadinessProbe)
	})

	t.Run("partial installation overrides with server fallback", func(t *testing.T) {
		serverLivenessOverride := &corev1.Probe{
			FailureThreshold:    5,
			SuccessThreshold:    1,
			InitialDelaySeconds: 30,
			PeriodSeconds:       10,
			TimeoutSeconds:      5,
		}

		serverReadinessOverride := &corev1.Probe{
			FailureThreshold:    3,
			SuccessThreshold:    1,
			InitialDelaySeconds: 20,
			PeriodSeconds:       8,
			TimeoutSeconds:      3,
		}

		// Installation only overrides liveness, readiness should fall back to server
		installationLivenessOverride := &corev1.Probe{
			FailureThreshold:    15,
			SuccessThreshold:    1,
			InitialDelaySeconds: 90,
			PeriodSeconds:       30,
			TimeoutSeconds:      20,
		}

		provisioner := Provisioner{
			params: ProvisioningParams{
				PodProbeOverrides: PodProbeOverrides{
					LivenessProbeOverride:  serverLivenessOverride,
					ReadinessProbeOverride: serverReadinessOverride,
				},
			},
		}

		installation := &model.Installation{
			PodProbeOverrides: &model.PodProbeOverrides{
				LivenessProbeOverride:  installationLivenessOverride,
				ReadinessProbeOverride: nil, // No installation override for readiness
			},
		}

		mattermost := &mmv1beta1.Mattermost{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-mattermost",
			},
			Spec: mmv1beta1.MattermostSpec{},
		}

		provisioner.ensurePodProbeOverrides(mattermost, installation)

		// Liveness should use installation override
		assert.Equal(t, *installationLivenessOverride, mattermost.Spec.Probes.LivenessProbe)
		// Readiness should fall back to server override
		assert.Equal(t, *serverReadinessOverride, mattermost.Spec.Probes.ReadinessProbe)
	})

	t.Run("installation with nil probe overrides uses server defaults", func(t *testing.T) {
		serverLivenessOverride := &corev1.Probe{
			FailureThreshold:    7,
			SuccessThreshold:    1,
			InitialDelaySeconds: 40,
			PeriodSeconds:       12,
			TimeoutSeconds:      6,
		}

		provisioner := Provisioner{
			params: ProvisioningParams{
				PodProbeOverrides: PodProbeOverrides{
					LivenessProbeOverride:  serverLivenessOverride,
					ReadinessProbeOverride: nil,
				},
			},
		}

		installation := &model.Installation{
			PodProbeOverrides: nil, // No installation overrides
		}

		mattermost := &mmv1beta1.Mattermost{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-mattermost",
			},
			Spec: mmv1beta1.MattermostSpec{},
		}

		provisioner.ensurePodProbeOverrides(mattermost, installation)

		// Should use server overrides
		assert.Equal(t, *serverLivenessOverride, mattermost.Spec.Probes.LivenessProbe)
		assert.Equal(t, corev1.Probe{}, mattermost.Spec.Probes.ReadinessProbe)
	})

	t.Run("installation with empty probe overrides uses server defaults", func(t *testing.T) {
		serverReadinessOverride := &corev1.Probe{
			FailureThreshold:    4,
			SuccessThreshold:    2,
			InitialDelaySeconds: 25,
			PeriodSeconds:       10,
			TimeoutSeconds:      4,
		}

		provisioner := Provisioner{
			params: ProvisioningParams{
				PodProbeOverrides: PodProbeOverrides{
					LivenessProbeOverride:  nil,
					ReadinessProbeOverride: serverReadinessOverride,
				},
			},
		}

		installation := &model.Installation{
			PodProbeOverrides: &model.PodProbeOverrides{
				LivenessProbeOverride:  nil,
				ReadinessProbeOverride: nil,
			},
		}

		mattermost := &mmv1beta1.Mattermost{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-mattermost",
			},
			Spec: mmv1beta1.MattermostSpec{},
		}

		provisioner.ensurePodProbeOverrides(mattermost, installation)

		// Should use server overrides since installation overrides are nil
		assert.Equal(t, corev1.Probe{}, mattermost.Spec.Probes.LivenessProbe)
		assert.Equal(t, *serverReadinessOverride, mattermost.Spec.Probes.ReadinessProbe)
	})

	t.Run("installation overrides only readiness, liveness falls back", func(t *testing.T) {
		serverLivenessOverride := &corev1.Probe{
			FailureThreshold:    6,
			SuccessThreshold:    1,
			InitialDelaySeconds: 35,
			PeriodSeconds:       12,
			TimeoutSeconds:      7,
		}

		installationReadinessOverride := &corev1.Probe{
			FailureThreshold:    9,
			SuccessThreshold:    3,
			InitialDelaySeconds: 55,
			PeriodSeconds:       18,
			TimeoutSeconds:      11,
		}

		provisioner := Provisioner{
			params: ProvisioningParams{
				PodProbeOverrides: PodProbeOverrides{
					LivenessProbeOverride:  serverLivenessOverride,
					ReadinessProbeOverride: nil,
				},
			},
		}

		installation := &model.Installation{
			PodProbeOverrides: &model.PodProbeOverrides{
				LivenessProbeOverride:  nil,
				ReadinessProbeOverride: installationReadinessOverride,
			},
		}

		mattermost := &mmv1beta1.Mattermost{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-mattermost",
			},
			Spec: mmv1beta1.MattermostSpec{},
		}

		provisioner.ensurePodProbeOverrides(mattermost, installation)

		// Liveness should fall back to server override
		assert.Equal(t, *serverLivenessOverride, mattermost.Spec.Probes.LivenessProbe)
		// Readiness should use installation override
		assert.Equal(t, *installationReadinessOverride, mattermost.Spec.Probes.ReadinessProbe)
	})

	t.Run("no server or installation overrides", func(t *testing.T) {
		provisioner := Provisioner{
			params: ProvisioningParams{
				PodProbeOverrides: PodProbeOverrides{}, // No server overrides
			},
		}

		installation := &model.Installation{
			PodProbeOverrides: nil, // No installation overrides
		}

		mattermost := &mmv1beta1.Mattermost{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-mattermost",
			},
			Spec: mmv1beta1.MattermostSpec{
				// Set some initial values to ensure they get cleared
				Probes: mmv1beta1.Probes{
					LivenessProbe: corev1.Probe{
						FailureThreshold: 999,
						TimeoutSeconds:   999,
					},
					ReadinessProbe: corev1.Probe{
						FailureThreshold: 888,
						TimeoutSeconds:   888,
					},
				},
			},
		}

		provisioner.ensurePodProbeOverrides(mattermost, installation)

		// Both probes should be cleared to empty Probe structs
		assert.Equal(t, corev1.Probe{}, mattermost.Spec.Probes.LivenessProbe)
		assert.Equal(t, corev1.Probe{}, mattermost.Spec.Probes.ReadinessProbe)
	})
}
