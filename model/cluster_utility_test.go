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
	assert.Equal(t, &HelmUtilityVersion{Chart: "0.9"}, u.Nginx)

	setUtilityVersion(u, "an_error", &HelmUtilityVersion{Chart: "9"})
	assert.Equal(t, &HelmUtilityVersion{Chart: "0.9"}, u.Nginx)
}

func TestGetUtilityVersion(t *testing.T) {
	u := UtilityGroupVersions{
		PrometheusOperator:  &HelmUtilityVersion{Chart: "3"},
		Thanos:              &HelmUtilityVersion{Chart: "4"},
		Nginx:               &HelmUtilityVersion{Chart: "5"},
		Fluentbit:           &HelmUtilityVersion{Chart: "6"},
		NodeProblemDetector: &HelmUtilityVersion{Chart: "7"},
		MetricsServer:       &HelmUtilityVersion{Chart: "8"},
		Velero:              &HelmUtilityVersion{Chart: "9"},
		Cloudprober:         &HelmUtilityVersion{Chart: "10"},
		Haproxy:             &HelmUtilityVersion{Chart: "11"},
	}

	assert.Equal(t, &HelmUtilityVersion{Chart: "3"}, getUtilityVersion(u, PrometheusOperatorCanonicalName))
	assert.Equal(t, &HelmUtilityVersion{Chart: "4"}, getUtilityVersion(u, ThanosCanonicalName))
	assert.Equal(t, &HelmUtilityVersion{Chart: "5"}, getUtilityVersion(u, NginxCanonicalName))
	assert.Equal(t, &HelmUtilityVersion{Chart: "6"}, getUtilityVersion(u, FluentbitCanonicalName))
	assert.Equal(t, &HelmUtilityVersion{Chart: "7"}, getUtilityVersion(u, NodeProblemDetectorCanonicalName))
	assert.Equal(t, &HelmUtilityVersion{Chart: "8"}, getUtilityVersion(u, MetricsServerCanonicalName))
	assert.Equal(t, &HelmUtilityVersion{Chart: "9"}, getUtilityVersion(u, VeleroCanonicalName))
	assert.Equal(t, &HelmUtilityVersion{Chart: "10"}, getUtilityVersion(u, CloudproberCanonicalName))
	assert.Equal(t, &HelmUtilityVersion{Chart: "11"}, getUtilityVersion(u, HaproxyCanonicalName))
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
	for _, testCase := range []struct {
		description             string
		currentMetadata         *UtilityMetadata
		desiredVersions         map[string]*HelmUtilityVersion
		expectedDesiredVersions UtilityGroupVersions
	}{
		{
			description:     "set desired utility without actual",
			currentMetadata: nil,
			desiredVersions: map[string]*HelmUtilityVersion{
				NginxCanonicalName: {Chart: "1.9.9", ValuesPath: "vals"},
			},
			expectedDesiredVersions: UtilityGroupVersions{
				Nginx: &HelmUtilityVersion{Chart: "1.9.9", ValuesPath: "vals"},
			},
		},
		{
			description: "set single desired utility, don't inherit from actual",
			currentMetadata: &UtilityMetadata{
				ActualVersions: UtilityGroupVersions{
					PrometheusOperator: &HelmUtilityVersion{ValuesPath: "prom", Chart: "1.0.0"},
					Nginx:              &HelmUtilityVersion{ValuesPath: "nginx", Chart: "2.0.0"},
				},
			},
			desiredVersions: map[string]*HelmUtilityVersion{
				NginxCanonicalName: {Chart: "3.0.0", ValuesPath: "nginx"},
			},
			expectedDesiredVersions: UtilityGroupVersions{
				Nginx: &HelmUtilityVersion{ValuesPath: "nginx", Chart: "3.0.0"},
			},
		},
		{
			description: "use all actual to override current desired",
			currentMetadata: &UtilityMetadata{
				ActualVersions: UtilityGroupVersions{
					PrometheusOperator: &HelmUtilityVersion{ValuesPath: "prom", Chart: "1.0.0"},
					Nginx:              &HelmUtilityVersion{ValuesPath: "nginx", Chart: "2.0.0"},
					Teleport:           &HelmUtilityVersion{ValuesPath: "teleport-kube-agent", Chart: "5.0.0"},
				},
				DesiredVersions: UtilityGroupVersions{
					PrometheusOperator: &HelmUtilityVersion{ValuesPath: "desired-prometheus", Chart: "0.1"},
					Nginx:              &HelmUtilityVersion{ValuesPath: "desired-nginx", Chart: "120.0.0"},
					Teleport:           &HelmUtilityVersion{ValuesPath: "desired-teleport-kube-agent", Chart: "15.0.0"},
				},
			},
			desiredVersions: nil,
			expectedDesiredVersions: UtilityGroupVersions{
				PrometheusOperator: &HelmUtilityVersion{ValuesPath: "desired-prometheus", Chart: "0.1"},
				Nginx:              &HelmUtilityVersion{ValuesPath: "desired-nginx", Chart: "120.0.0"},
				Teleport:           &HelmUtilityVersion{ValuesPath: "desired-teleport-kube-agent", Chart: "15.0.0"},
			},
		},
	} {
		t.Run(testCase.description, func(t *testing.T) {
			c := &Cluster{
				UtilityMetadata: testCase.currentMetadata,
			}
			c.SetUtilityDesiredVersions(testCase.desiredVersions)
			assert.Equal(t, testCase.expectedDesiredVersions, c.UtilityMetadata.DesiredVersions)
		})
	}
}

func TestGetActualVersion(t *testing.T) {
	c := &Cluster{
		UtilityMetadata: &UtilityMetadata{
			DesiredVersions: UtilityGroupVersions{
				PrometheusOperator:  &HelmUtilityVersion{Chart: ""},
				Thanos:              &HelmUtilityVersion{Chart: ""},
				Nginx:               &HelmUtilityVersion{Chart: "10.3"},
				Fluentbit:           &HelmUtilityVersion{Chart: "1337"},
				Teleport:            &HelmUtilityVersion{Chart: "12345"},
				Pgbouncer:           &HelmUtilityVersion{Chart: "123456"},
				Promtail:            &HelmUtilityVersion{Chart: "123456"},
				Rtcd:                &HelmUtilityVersion{Chart: "123456"},
				Kubecost:            &HelmUtilityVersion{Chart: "12345678"},
				NodeProblemDetector: &HelmUtilityVersion{Chart: "123456789"},
				MetricsServer:       &HelmUtilityVersion{Chart: "1234567899"},
				Velero:              &HelmUtilityVersion{Chart: "12345678910"},
				Cloudprober:         &HelmUtilityVersion{Chart: "123456789101"},
				Haproxy:             &HelmUtilityVersion{Chart: "123456789111"},
			},
			ActualVersions: UtilityGroupVersions{
				PrometheusOperator:  &HelmUtilityVersion{Chart: "kube-prometheus-stack-9.4"},
				Thanos:              &HelmUtilityVersion{Chart: "thanos-2.4"},
				Nginx:               &HelmUtilityVersion{Chart: "nginx-10.2"},
				Fluentbit:           &HelmUtilityVersion{Chart: "fluent-bit-0.9"},
				Teleport:            &HelmUtilityVersion{Chart: "teleport-kube-agent-6.2.8"},
				Pgbouncer:           &HelmUtilityVersion{Chart: "pgbouncer-1.2.0"},
				Promtail:            &HelmUtilityVersion{Chart: "promtail-6.2.2"},
				Rtcd:                &HelmUtilityVersion{Chart: "rtcd-1.1.0"},
				Kubecost:            &HelmUtilityVersion{Chart: "cost-analyzer-1.95.0"},
				NodeProblemDetector: &HelmUtilityVersion{Chart: "node-problem-detector-2.0.5"},
				MetricsServer:       &HelmUtilityVersion{Chart: "metrics-server-3.8.2"},
				Velero:              &HelmUtilityVersion{Chart: "velero-2.31.3"},
				Cloudprober:         &HelmUtilityVersion{Chart: "cloudprober-0.1.0"},
				Haproxy:             &HelmUtilityVersion{Chart: "kubernetes-ingress-1.24.0"},
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
	assert.Equal(t, &HelmUtilityVersion{Chart: "teleport-kube-agent-6.2.8"}, version)

	version = c.ActualUtilityVersion(PgbouncerCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "pgbouncer-1.2.0"}, version)

	version = c.ActualUtilityVersion(PromtailCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "promtail-6.2.2"}, version)

	version = c.ActualUtilityVersion(RtcdCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "rtcd-1.1.0"}, version)

	version = c.ActualUtilityVersion(KubecostCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "cost-analyzer-1.95.0"}, version)

	version = c.ActualUtilityVersion(NodeProblemDetectorCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "node-problem-detector-2.0.5"}, version)

	version = c.ActualUtilityVersion(MetricsServerCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "metrics-server-3.8.2"}, version)

	version = c.ActualUtilityVersion(VeleroCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "velero-2.31.3"}, version)

	version = c.ActualUtilityVersion(CloudproberCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "cloudprober-0.1.0"}, version)

	version = c.ActualUtilityVersion(HaproxyCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "kubernetes-ingress-1.24.0"}, version)

	version = c.ActualUtilityVersion("something else that doesn't exist")
	assert.Equal(t, version, nilHuv)
}

func TestGetDesiredVersion(t *testing.T) {
	c := &Cluster{
		UtilityMetadata: &UtilityMetadata{
			DesiredVersions: UtilityGroupVersions{
				PrometheusOperator:  &HelmUtilityVersion{Chart: ""},
				Thanos:              &HelmUtilityVersion{Chart: ""},
				Nginx:               &HelmUtilityVersion{Chart: "10.3"},
				Fluentbit:           &HelmUtilityVersion{Chart: "1337"},
				Teleport:            &HelmUtilityVersion{Chart: "12345"},
				Pgbouncer:           &HelmUtilityVersion{Chart: "123456"},
				Promtail:            &HelmUtilityVersion{Chart: "123456"},
				Rtcd:                &HelmUtilityVersion{Chart: "123456"},
				Kubecost:            &HelmUtilityVersion{Chart: "12345678"},
				NodeProblemDetector: &HelmUtilityVersion{Chart: "123456789"},
				MetricsServer:       &HelmUtilityVersion{Chart: "1234567899"},
				Velero:              &HelmUtilityVersion{Chart: "12345678910"},
				Cloudprober:         &HelmUtilityVersion{Chart: "123456789101"},
				Haproxy:             &HelmUtilityVersion{Chart: "123456789111"},
			},
			ActualVersions: UtilityGroupVersions{
				PrometheusOperator:  &HelmUtilityVersion{Chart: "kube-prometheus-stack-9.4"},
				Thanos:              &HelmUtilityVersion{Chart: "thanos-2.4"},
				Nginx:               &HelmUtilityVersion{Chart: "nginx-10.2"},
				Fluentbit:           &HelmUtilityVersion{Chart: "fluent-bit-0.9"},
				Teleport:            &HelmUtilityVersion{Chart: "teleport-kube-agent-6.2.8"},
				Pgbouncer:           &HelmUtilityVersion{Chart: "pgbouncer-1.2.0"},
				Promtail:            &HelmUtilityVersion{Chart: "promtail-6.2.2"},
				Rtcd:                &HelmUtilityVersion{Chart: "rtcd-1.1.0"},
				Kubecost:            &HelmUtilityVersion{Chart: "cost-analyzer-1.95.0"},
				NodeProblemDetector: &HelmUtilityVersion{Chart: "node-problem-detector-2.0.5"},
				MetricsServer:       &HelmUtilityVersion{Chart: "metrics-server-3.8.2"},
				Velero:              &HelmUtilityVersion{Chart: "velero-2.31.3"},
				Cloudprober:         &HelmUtilityVersion{Chart: "cloudprober-0.1.0"},
				Haproxy:             &HelmUtilityVersion{Chart: "kubernetes-ingress-1.24.0"},
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

	version = c.DesiredUtilityVersion(PgbouncerCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "123456"}, version)

	version = c.DesiredUtilityVersion(PromtailCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "123456"}, version)

	version = c.DesiredUtilityVersion(RtcdCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "123456"}, version)

	version = c.DesiredUtilityVersion(KubecostCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "12345678"}, version)

	version = c.DesiredUtilityVersion(NodeProblemDetectorCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "123456789"}, version)

	version = c.DesiredUtilityVersion(MetricsServerCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "1234567899"}, version)

	version = c.DesiredUtilityVersion(VeleroCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "12345678910"}, version)

	version = c.DesiredUtilityVersion(CloudproberCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "123456789101"}, version)

	version = c.DesiredUtilityVersion(HaproxyCanonicalName)
	assert.Equal(t, &HelmUtilityVersion{Chart: "123456789111"}, version)

	version = c.DesiredUtilityVersion("something else that doesn't exist")
	assert.Equal(t, nilHuv, version)
}
