// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/provisioner"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
)

func TestGenerateProbeOverrides(t *testing.T) {
	t.Run("no probe flags changed", func(t *testing.T) {
		flags := &serverFlags{}
		result := flags.generateProbeOverrides()

		expected := provisioner.PodProbeOverrides{}
		assert.Equal(t, expected, result)
		assert.Nil(t, result.LivenessProbeOverride)
		assert.Nil(t, result.ReadinessProbeOverride)
	})

	t.Run("only liveness probe flags changed", func(t *testing.T) {
		flags := &serverFlags{
			provisioningParams: provisioningParams{
				probeLivenessFailureThreshold:    5,
				probeLivenessSuccessThreshold:    2,
				probeLivenessInitialDelaySeconds: 60,
				probeLivenessPeriodSeconds:       15,
				probeLivenessTimeoutSeconds:      10,
			},
			serverFlagChanged: serverFlagChanged{
				probeLivenessFailureThresholdChanged:    true,
				probeLivenessSuccessThresholdChanged:    true,
				probeLivenessInitialDelaySecondsChanged: true,
				probeLivenessPeriodSecondsChanged:       true,
				probeLivenessTimeoutSecondsChanged:      true,
			},
		}

		result := flags.generateProbeOverrides()

		assert.NotNil(t, result.LivenessProbeOverride)
		assert.Nil(t, result.ReadinessProbeOverride)

		expected := &corev1.Probe{
			FailureThreshold:    5,
			SuccessThreshold:    2,
			InitialDelaySeconds: 60,
			PeriodSeconds:       15,
			TimeoutSeconds:      10,
		}
		assert.Equal(t, expected, result.LivenessProbeOverride)
	})

	t.Run("only readiness probe flags changed", func(t *testing.T) {
		flags := &serverFlags{
			provisioningParams: provisioningParams{
				probeReadinessFailureThreshold:    3,
				probeReadinessSuccessThreshold:    1,
				probeReadinessInitialDelaySeconds: 45,
				probeReadinessPeriodSeconds:       20,
				probeReadinessTimeoutSeconds:      8,
			},
			serverFlagChanged: serverFlagChanged{
				probeReadinessFailureThresholdChanged:    true,
				probeReadinessSuccessThresholdChanged:    true,
				probeReadinessInitialDelaySecondsChanged: true,
				probeReadinessPeriodSecondsChanged:       true,
				probeReadinessTimeoutSecondsChanged:      true,
			},
		}

		result := flags.generateProbeOverrides()

		assert.Nil(t, result.LivenessProbeOverride)
		assert.NotNil(t, result.ReadinessProbeOverride)

		expected := &corev1.Probe{
			FailureThreshold:    3,
			SuccessThreshold:    1,
			InitialDelaySeconds: 45,
			PeriodSeconds:       20,
			TimeoutSeconds:      8,
		}
		assert.Equal(t, expected, result.ReadinessProbeOverride)
	})

	t.Run("both liveness and readiness probe flags changed", func(t *testing.T) {
		flags := &serverFlags{
			provisioningParams: provisioningParams{
				probeLivenessFailureThreshold:     4,
				probeLivenessSuccessThreshold:     1,
				probeLivenessInitialDelaySeconds:  30,
				probeLivenessPeriodSeconds:        5,
				probeLivenessTimeoutSeconds:       3,
				probeReadinessFailureThreshold:    6,
				probeReadinessSuccessThreshold:    2,
				probeReadinessInitialDelaySeconds: 20,
				probeReadinessPeriodSeconds:       10,
				probeReadinessTimeoutSeconds:      7,
			},
			serverFlagChanged: serverFlagChanged{
				probeLivenessFailureThresholdChanged:     true,
				probeLivenessSuccessThresholdChanged:     true,
				probeLivenessInitialDelaySecondsChanged:  true,
				probeLivenessPeriodSecondsChanged:        true,
				probeLivenessTimeoutSecondsChanged:       true,
				probeReadinessFailureThresholdChanged:    true,
				probeReadinessSuccessThresholdChanged:    true,
				probeReadinessInitialDelaySecondsChanged: true,
				probeReadinessPeriodSecondsChanged:       true,
				probeReadinessTimeoutSecondsChanged:      true,
			},
		}

		result := flags.generateProbeOverrides()

		assert.NotNil(t, result.LivenessProbeOverride)
		assert.NotNil(t, result.ReadinessProbeOverride)

		expectedLiveness := &corev1.Probe{
			FailureThreshold:    4,
			SuccessThreshold:    1,
			InitialDelaySeconds: 30,
			PeriodSeconds:       5,
			TimeoutSeconds:      3,
		}
		assert.Equal(t, expectedLiveness, result.LivenessProbeOverride)

		expectedReadiness := &corev1.Probe{
			FailureThreshold:    6,
			SuccessThreshold:    2,
			InitialDelaySeconds: 20,
			PeriodSeconds:       10,
			TimeoutSeconds:      7,
		}
		assert.Equal(t, expectedReadiness, result.ReadinessProbeOverride)
	})

	t.Run("partial liveness probe flags changed", func(t *testing.T) {
		flags := &serverFlags{
			provisioningParams: provisioningParams{
				probeLivenessFailureThreshold: 8,
				probeLivenessTimeoutSeconds:   12,
			},
			serverFlagChanged: serverFlagChanged{
				probeLivenessFailureThresholdChanged: true,
				probeLivenessTimeoutSecondsChanged:   true,
				// Other liveness flags not changed
			},
		}

		result := flags.generateProbeOverrides()

		assert.NotNil(t, result.LivenessProbeOverride)
		assert.Nil(t, result.ReadinessProbeOverride)

		expected := &corev1.Probe{
			FailureThreshold: 8,
			TimeoutSeconds:   12,
			// Other fields should be zero values since they weren't changed
		}
		assert.Equal(t, expected, result.LivenessProbeOverride)
	})

	t.Run("partial readiness probe flags changed", func(t *testing.T) {
		flags := &serverFlags{
			provisioningParams: provisioningParams{
				probeReadinessSuccessThreshold:    3,
				probeReadinessPeriodSeconds:       25,
				probeReadinessInitialDelaySeconds: 100,
			},
			serverFlagChanged: serverFlagChanged{
				probeReadinessSuccessThresholdChanged:    true,
				probeReadinessPeriodSecondsChanged:       true,
				probeReadinessInitialDelaySecondsChanged: true,
				// Other readiness flags not changed
			},
		}

		result := flags.generateProbeOverrides()

		assert.Nil(t, result.LivenessProbeOverride)
		assert.NotNil(t, result.ReadinessProbeOverride)

		expected := &corev1.Probe{
			SuccessThreshold:    3,
			PeriodSeconds:       25,
			InitialDelaySeconds: 100,
			// Other fields should be zero values since they weren't changed
		}
		assert.Equal(t, expected, result.ReadinessProbeOverride)
	})

	t.Run("single liveness flag changed", func(t *testing.T) {
		flags := &serverFlags{
			provisioningParams: provisioningParams{
				probeLivenessFailureThreshold: 15,
			},
			serverFlagChanged: serverFlagChanged{
				probeLivenessFailureThresholdChanged: true,
			},
		}

		result := flags.generateProbeOverrides()

		assert.NotNil(t, result.LivenessProbeOverride)
		assert.Nil(t, result.ReadinessProbeOverride)

		expected := &corev1.Probe{
			FailureThreshold: 15,
		}
		assert.Equal(t, expected, result.LivenessProbeOverride)
	})

	t.Run("single readiness flag changed", func(t *testing.T) {
		flags := &serverFlags{
			provisioningParams: provisioningParams{
				probeReadinessInitialDelaySeconds: 200,
			},
			serverFlagChanged: serverFlagChanged{
				probeReadinessInitialDelaySecondsChanged: true,
			},
		}

		result := flags.generateProbeOverrides()

		assert.Nil(t, result.LivenessProbeOverride)
		assert.NotNil(t, result.ReadinessProbeOverride)

		expected := &corev1.Probe{
			InitialDelaySeconds: 200,
		}
		assert.Equal(t, expected, result.ReadinessProbeOverride)
	})
}
