// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetUtilityVersion(t *testing.T) {
	u := &utilityVersions{
		Prometheus:         "",
		PrometheusOperator: "",
		Nginx:              "",
		Fluentbit:          "",
	}

	setUtilityVersion(u, PrometheusCanonicalName, "1.9")
	assert.Equal(t, u.Prometheus, "1.9")

	setUtilityVersion(u, NginxCanonicalName, "0.9")
	assert.Equal(t, u.Prometheus, "1.9")
	assert.Equal(t, u.Nginx, "0.9")

	setUtilityVersion(u, "an_error", "9")
	assert.Equal(t, u.Prometheus, "1.9")
	assert.Equal(t, u.Nginx, "0.9")
}

func TestGetUtilityVersion(t *testing.T) {
	u := &utilityVersions{
		Prometheus:         "4",
		PrometheusOperator: "3",
		Nginx:              "5",
		Fluentbit:          "6",
	}

	assert.Equal(t, getUtilityVersion(u, PrometheusCanonicalName), "4")
	assert.Equal(t, getUtilityVersion(u, PrometheusOperatorCanonicalName), "3")
	assert.Equal(t, getUtilityVersion(u, NginxCanonicalName), "5")
	assert.Equal(t, getUtilityVersion(u, FluentbitCanonicalName), "6")
	assert.Equal(t, getUtilityVersion(u, "anything else"), "")
}

func TestSetActualVersion(t *testing.T) {
	c := &Cluster{}

	assert.Nil(t, c.UtilityMetadata)
	err := c.SetUtilityActualVersion(NginxCanonicalName, "1.9.9")
	require.NoError(t, err)
	assert.NotNil(t, c.UtilityMetadata)
	version, err := c.ActualUtilityVersion(NginxCanonicalName)
	require.NoError(t, err)
	assert.Equal(t, "1.9.9", version)
}

func TestSetDesired(t *testing.T) {
	c := &Cluster{}

	assert.Nil(t, c.UtilityMetadata)
	err := c.SetUtilityDesiredVersions(map[string]string{
		NginxCanonicalName: "1.9.9",
	})
	require.NoError(t, err)

	assert.NotNil(t, c.UtilityMetadata)

	version, err := c.DesiredUtilityVersion(NginxCanonicalName)
	require.NoError(t, err)
	assert.Equal(t, "1.9.9", version)

	version, err = c.DesiredUtilityVersion(PrometheusCanonicalName)
	require.NoError(t, err)
	assert.Equal(t, "", version)

	version, err = c.DesiredUtilityVersion(PrometheusOperatorCanonicalName)
	require.NoError(t, err)
	assert.Equal(t, "", version)
}

func TestGetActualVersion(t *testing.T) {
	c := &Cluster{
		UtilityMetadata: &UtilityMetadata{
			DesiredVersions: utilityVersions{
				Prometheus:         "",
				PrometheusOperator: "",
				Nginx:              "10.3",
				Fluentbit:          "1337",
				Teleport:           "12345",
			},
			ActualVersions: utilityVersions{
				Prometheus:         "prometheus-10.3",
				PrometheusOperator: "kube-prometheus-stack-9.4",
				Nginx:              "nginx-10.2",
				Fluentbit:          "fluent-bit-0.9",
				Teleport:           "teleport-0.3.0",
			},
		},
	}

	version, err := c.ActualUtilityVersion(PrometheusCanonicalName)
	assert.NoError(t, err)
	assert.Equal(t, "prometheus-10.3", version)

	version, err = c.ActualUtilityVersion(PrometheusOperatorCanonicalName)
	assert.NoError(t, err)
	assert.Equal(t, "kube-prometheus-stack-9.4", version)

	version, err = c.ActualUtilityVersion(NginxCanonicalName)
	assert.NoError(t, err)
	assert.Equal(t, "nginx-10.2", version)

	version, err = c.ActualUtilityVersion(FluentbitCanonicalName)
	assert.NoError(t, err)
	assert.Equal(t, "fluent-bit-0.9", version)

	version, err = c.ActualUtilityVersion(TeleportCanonicalName)
	assert.NoError(t, err)
	assert.Equal(t, "teleport-0.3.0", version)

	version, err = c.ActualUtilityVersion("something else that doesn't exist")
	assert.NoError(t, err)
	assert.Equal(t, "", version)
}

func TestGetDesiredVersion(t *testing.T) {
	c := &Cluster{
		UtilityMetadata: &UtilityMetadata{
			DesiredVersions: utilityVersions{
				Prometheus:         "",
				PrometheusOperator: "",
				Nginx:              "10.3",
				Fluentbit:          "1337",
				Teleport:           "12345",
			},
			ActualVersions: utilityVersions{
				Prometheus:         "prometheus-10.3",
				PrometheusOperator: "kube-prometheus-stack-9.4",
				Nginx:              "nginx-10.2",
				Fluentbit:          "fluent-bit-0.9",
				Teleport:           "teleport-0.3.0",
			},
		},
	}

	version, err := c.DesiredUtilityVersion(PrometheusCanonicalName)
	assert.NoError(t, err)
	assert.Equal(t, "", version)

	version, err = c.DesiredUtilityVersion(PrometheusOperatorCanonicalName)
	assert.NoError(t, err)
	assert.Equal(t, "", version)

	version, err = c.DesiredUtilityVersion(NginxCanonicalName)
	assert.NoError(t, err)
	assert.Equal(t, "10.3", version)

	version, err = c.DesiredUtilityVersion(FluentbitCanonicalName)
	assert.NoError(t, err)
	assert.Equal(t, "1337", version)

	version, err = c.DesiredUtilityVersion(TeleportCanonicalName)
	assert.NoError(t, err)
	assert.Equal(t, "12345", version)

	version, err = c.DesiredUtilityVersion("something else that doesn't exist")
	assert.NoError(t, err)
	assert.Equal(t, "", version)
}
