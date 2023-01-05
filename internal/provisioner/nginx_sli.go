// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"
	"time"

	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getNginxSlothObjectName(clusterInstallation *model.ClusterInstallation) string {
	return fmt.Sprintf("slo-nginx-my-enterpise-%s", clusterInstallation.InstallationID)
}

func makeNginxSLIs(clusterInstallation *model.ClusterInstallation) slothv1.PrometheusServiceLevel {
	pslName := getNginxSlothObjectName(clusterInstallation)
	serviceName := makeClusterInstallationName(clusterInstallation)

	sli := slothv1.PrometheusServiceLevel{
		ObjectMeta: metav1.ObjectMeta{
			Name: pslName,
			Labels: map[string]string{
				"app":     "kube-prometheus-stack",
				"release": "prometheus-operator",
			},
			Namespace: prometheusNamespace,
		},
		Spec: slothv1.PrometheusServiceLevelSpec{
			Service: fmt.Sprintf("nginx-%s-service", clusterInstallation.InstallationID),
			Labels: map[string]string{
				"owner": "sre-team",
			},
			SLOs: []slothv1.SLO{
				{
					Name:        "requests-availability",
					Objective:   99.9,
					Description: "Common SLO based on availability for HTTP request responses measured on ingress layer.",
					SLI: slothv1.SLI{Events: &slothv1.SLIEvents{
						ErrorQuery: "sum(rate(nginx_ingress_controller_request_duration_seconds_count{exported_service='" + serviceName + "',status=~'(5..|429|499)'}[{{.window}}]))",
						TotalQuery: "sum(rate(nginx_ingress_controller_request_duration_seconds_count{exported_service='" + serviceName + "'}[{{.window}}]))",
					}},
					Alerting: slothv1.Alerting{
						PageAlert:   slothv1.Alert{Disable: true},
						TicketAlert: slothv1.Alert{Disable: true},
					},
				}},
		},
	}

	return sli
}

func createOrUpdateNginxSLIs(clusterInstallation *model.ClusterInstallation, k8sClient *k8s.KubeClient, logger log.FieldLogger) error {
	sli := makeNginxSLIs(clusterInstallation)
	return createOrUpdateClusterPrometheusServiceLevel(sli, k8sClient, logger)
}

func deleteNginxSLI(clusterInstallation *model.ClusterInstallation, k8sClient *k8s.KubeClient, logger log.FieldLogger) error {
	pslName := getNginxSlothObjectName(clusterInstallation)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*60)
	defer cancel()
	_, err := k8sClient.SlothClientsetV1.SlothV1().PrometheusServiceLevels(prometheusNamespace).Get(ctx, pslName, metav1.GetOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		logger.Debugf("sloth CRD doesn't exist on cluster: %s", err)
		return nil
	}
	err = k8sClient.SlothClientsetV1.SlothV1().PrometheusServiceLevels(prometheusNamespace).Delete(ctx, pslName, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to delete enterprise nginx sli")
	}
	return nil
}
