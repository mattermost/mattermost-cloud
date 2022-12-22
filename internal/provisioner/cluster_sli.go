// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"strings"

	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	slothServiceLevelTypeLabel        = "serviceLevelType"
	slothServiceLevelTypeClusterValue = "cluster"
	slothServiceLevelTypeRingValue    = "ring"
)

func makeRingSLOName(group *model.GroupDTO) string {
	return strings.ToLower(group.Name) + "-ring-" + group.ID
}

func makeRingSLOs(group *model.GroupDTO, objective float64) slothv1.PrometheusServiceLevel {
	resourceName := makeRingSLOName(group)
	sli := slothv1.PrometheusServiceLevel{
		ObjectMeta: metav1.ObjectMeta{
			Name: resourceName,
			Labels: map[string]string{
				"app":                      "kube-prometheus-stack",
				"release":                  "prometheus-operator",
				slothServiceLevelTypeLabel: slothServiceLevelTypeRingValue,
			},
		},
		Spec: slothv1.PrometheusServiceLevelSpec{
			Labels: map[string]string{
				"owner": "sreteam",
			},
			Service: resourceName,
			SLOs: []slothv1.SLO{
				{
					Alerting: slothv1.Alerting{
						PageAlert:   slothv1.Alert{Disable: true},
						TicketAlert: slothv1.Alert{Disable: true},
					},
					Description: "Availability metric for mattermost API on " + group.Name,
					Name:        "requests-availability-cluster",
					Objective:   objective,
					SLI: slothv1.SLI{Events: &slothv1.SLIEvents{
						ErrorQuery: "sum(rate(mattermost_api_time_count{installationGroupId='" + group.ID + "',status_code=~'(5..|429|499)'}[{{.window}}]))",
						TotalQuery: "sum(rate(mattermost_api_time_count{installationGroupId='" + group.ID + "'}[{{.window}}]))",
					}},
				}},
		},
	}

	return sli
}

func makeClusterSLOs(cluster *model.Cluster, objective float64) slothv1.PrometheusServiceLevel {
	resourceName := "cluster-" + cluster.ID
	sli := slothv1.PrometheusServiceLevel{
		ObjectMeta: metav1.ObjectMeta{
			Name: resourceName,
			Labels: map[string]string{
				"app":                      "kube-prometheus-stack",
				"release":                  "prometheus-operator",
				slothServiceLevelTypeLabel: slothServiceLevelTypeClusterValue,
			},
		},
		Spec: slothv1.PrometheusServiceLevelSpec{
			Labels: map[string]string{
				"owner": "sreteam",
			},
			Service: resourceName,
			SLOs: []slothv1.SLO{
				{
					Alerting: slothv1.Alerting{
						PageAlert:   slothv1.Alert{Disable: true},
						TicketAlert: slothv1.Alert{Disable: true},
					},
					Description: "Availability metric for mattermost API on cluster " + cluster.ID,
					Name:        "requests-availability-cluster",
					Objective:   objective,
					SLI: slothv1.SLI{Events: &slothv1.SLIEvents{
						ErrorQuery: "sum(rate(mattermost_api_time_count{clusterID='" + cluster.ID + "',status_code=~'(5..|429|499)'}[{{.window}}]))",
						TotalQuery: "sum(rate(mattermost_api_time_count{clusterID='" + cluster.ID + "'}[{{.window}}]))",
					}},
				}},
		},
	}

	return sli
}

func createOrUpdateClusterSLOs(cluster *model.Cluster, k8sClient *k8s.KubeClient, objective float64, logger log.FieldLogger) error {
	sli := makeClusterSLOs(cluster, objective)
	return createOrUpdateClusterPrometheusServiceLevel(sli, k8sClient, logger)
}

func createOrUpdateRingSLOs(group *model.GroupDTO, k8sClient *k8s.KubeClient, objective float64, logger log.FieldLogger) error {
	sli := makeRingSLOs(group, objective)
	return createOrUpdateClusterPrometheusServiceLevel(sli, k8sClient, logger)
}
