// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var nilHuv *HelmUtilityVersion // provides a typed nil; never initialize this

func TestSetUtilityVersion(t *testing.T) {
	u := &UtilityGroupVersions{
		PrometheusOperator: &HelmUtilityVersion{Chart: ""},
		Nginx:              &HelmUtilityVersion{Chart: ""},
		Fluentbit:          &HelmUtilityVersion{Chart: ""},
	}

	setUtilityVersion(u, NginxCanonicalName, &HelmUtilityVersion{Chart: "0.9"})
	assert.Equal(t, u.Nginx, &HelmUtilityVersion{Chart: "0.9"})

	setUtilityVersion(u, "an_error", &HelmUtilityVersion{Chart: "9"})
	assert.Equal(t, u.Nginx, &HelmUtilityVersion{Chart: "0.9"})
}

func TestGetUtilityVersion(t *testing.T) {
	u := UtilityGroupVersions{
		PrometheusOperator: &HelmUtilityVersion{Chart: "3"},
		Thanos:             &HelmUtilityVersion{Chart: "4"},
		Nginx:              &HelmUtilityVersion{Chart: "5"},
		Fluentbit:          &HelmUtilityVersion{Chart: "6"},
	}

	assert.Equal(t, &HelmUtilityVersion{Chart: "3"}, getUtilityVersion(u, PrometheusOperatorCanonicalName))
	assert.Equal(t, &HelmUtilityVersion{Chart: "4"}, getUtilityVersion(u, ThanosCanonicalName))
	assert.Equal(t, &HelmUtilityVersion{Chart: "5"}, getUtilityVersion(u, NginxCanonicalName))
	assert.Equal(t, &HelmUtilityVersion{Chart: "6"}, getUtilityVersion(u, FluentbitCanonicalName))
	assert.Equal(t, nilHuv, getUtilityVersion(u, "anything else"))
}

func TestSetActualVersion(t *testing.T) {
	c := &Cluster{}

	assert.Nil(t, c.UtilityMetadata)
	err := c.SetUtilityActualVersion(NginxCanonicalName, &HelmUtilityVersion{Chart: "1.9.9"})
	require.NoError(t, err)
	assert.NotNil(t, c.UtilityMetadata)
	version := c.ActualUtilityVersion(NginxCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "1.9.9"}, version)
}

func TestSetDesired(t *testing.T) {
	c := &Cluster{}

	assert.Nil(t, c.UtilityMetadata)
	err := c.SetUtilityDesiredVersions(map[string]*HelmUtilityVersion{
		NginxCanonicalName: {Chart: "1.9.9"},
	})
	require.NoError(t, err)

	assert.NotNil(t, c.UtilityMetadata)

	version := c.DesiredUtilityVersion(NginxCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "1.9.9"}, version)

	var nilVersion *HelmUtilityVersion = nil
	version = c.DesiredUtilityVersion(PrometheusOperatorCanonicalName)
	require.NoError(t, err)
	assert.Equal(t, nilVersion, version)

	version = c.DesiredUtilityVersion(ThanosCanonicalName)
	assert.Equal(t, nilVersion, version)
}

func TestGetActualVersion(t *testing.T) {
	c := &Cluster{
		UtilityMetadata: &UtilityMetadata{
			DesiredVersions: UtilityGroupVersions{
				PrometheusOperator: &HelmUtilityVersion{Chart: ""},
				Thanos:             &HelmUtilityVersion{Chart: ""},
				Nginx:              &HelmUtilityVersion{Chart: "10.3"},
				Fluentbit:          &HelmUtilityVersion{Chart: "1337"},
				Teleport:           &HelmUtilityVersion{Chart: "12345"},
			},
			ActualVersions: UtilityGroupVersions{
				PrometheusOperator: &HelmUtilityVersion{Chart: "kube-prometheus-stack-9.4"},
				Thanos:             &HelmUtilityVersion{Chart: "thanos-2.4"},
				Nginx:              &HelmUtilityVersion{Chart: "nginx-10.2"},
				Fluentbit:          &HelmUtilityVersion{Chart: "fluent-bit-0.9"},
				Teleport:           &HelmUtilityVersion{Chart: "teleport-0.3.0"},
			},
		},
	}

	version := c.ActualUtilityVersion(PrometheusOperatorCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "kube-prometheus-stack-9.4"}, version)

	version = c.ActualUtilityVersion(ThanosCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "thanos-2.4"}, version)

	version = c.ActualUtilityVersion(NginxCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "nginx-10.2"}, version)

	version = c.ActualUtilityVersion(FluentbitCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "fluent-bit-0.9"}, version)

	version = c.ActualUtilityVersion(TeleportCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "teleport-0.3.0"}, version)

	version = c.ActualUtilityVersion("something else that doesn't exist")
	assert.Equal(t, version, nilHuv)
}

func TestGetDesiredVersion(t *testing.T) {
	c := &Cluster{
		UtilityMetadata: &UtilityMetadata{
			DesiredVersions: UtilityGroupVersions{
				PrometheusOperator: &HelmUtilityVersion{Chart: ""},
				Thanos:             &HelmUtilityVersion{Chart: ""},
				Nginx:              &HelmUtilityVersion{Chart: "10.3"},
				Fluentbit:          &HelmUtilityVersion{Chart: "1337"},
				Teleport:           &HelmUtilityVersion{Chart: "12345"},
			},
			ActualVersions: UtilityGroupVersions{
				PrometheusOperator: &HelmUtilityVersion{Chart: "kube-prometheus-stack-9.4"},
				Thanos:             &HelmUtilityVersion{Chart: "thanos-2.4"},
				Nginx:              &HelmUtilityVersion{Chart: "nginx-10.2"},
				Fluentbit:          &HelmUtilityVersion{Chart: "fluent-bit-0.9"},
				Teleport:           &HelmUtilityVersion{Chart: "teleport-0.3.0"},
			},
		},
	}

	version := c.DesiredUtilityVersion(PrometheusOperatorCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: ""}, version)

	version = c.DesiredUtilityVersion(ThanosCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: ""}, version)

	version = c.DesiredUtilityVersion(NginxCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "10.3"}, version)

	version = c.DesiredUtilityVersion(FluentbitCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "1337"}, version)

	version = c.DesiredUtilityVersion(TeleportCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "12345"}, version)

	version = c.DesiredUtilityVersion("something else that doesn't exist")
	assert.Equal(t, nilHuv, version)
}
