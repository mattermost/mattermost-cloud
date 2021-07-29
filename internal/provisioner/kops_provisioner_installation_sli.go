// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (provisioner *KopsProvisioner) createInstallationSLI(clusterInstallation *model.ClusterInstallation, k8sClient *k8s.KubeClient) error {

	sli := &slothv1.PrometheusServiceLevel{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterInstallation.InstallationID,
			Labels: map[string]string{
				"app":     "kube-prometheus-stack",
				"release": "prometheus-operator",
			},
		},
		Spec: slothv1.PrometheusServiceLevelSpec{
			Service: clusterInstallation.InstallationID,
			Labels: map[string]string{
				"owner": "sreteam",
			},
			SLOs: []slothv1.SLO{
				{
					Name:        "requests-availability",
					Objective:   99.9,
					Description: "Availability metric for mattermost API",
					SLI: slothv1.SLI{Events: &slothv1.SLIEvents{
						ErrorQuery: "sum(rate(mattermost_api_time_count{job='mm-" + clusterInstallation.Namespace[0:4] + "',code=~'(5..|429)'}[{{.window}}]))",
						TotalQuery: "sum(rate(mattermost_api_time_count{job='mm-" + clusterInstallation.Namespace[0:4] + "'}[{{.window}}]))",
					}},
					Alerting: slothv1.Alerting{
						Name: "mm-" + clusterInstallation.Namespace[0:4] + "-service-HighAPIErrorRate",
						Labels: map[string]string{
							"category": "availability",
						},
						Annotations: map[string]string{
							"summary": "High error rate on requests responses",
						},
					},
				}},
		},
	}

	ctx := context.TODO()
	_, err := k8sClient.SlothClientsetV1.SlothV1().PrometheusServiceLevels("prometheus").Create(ctx, sli, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to create cluster installation sli")
	}
	return nil
}

func (provisioner *KopsProvisioner) createIfNotExistInstallationSLI(clusterInstallation *model.ClusterInstallation, k8sClient *k8s.KubeClient) error {
	sli := &slothv1.PrometheusServiceLevel{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterInstallation.InstallationID,
			Labels: map[string]string{
				"app":     "kube-prometheus-stack",
				"release": "prometheus-operator",
			},
		},
		Spec: slothv1.PrometheusServiceLevelSpec{
			Service: clusterInstallation.InstallationID,
			Labels: map[string]string{
				"owner": "sreteam",
			},
			SLOs: []slothv1.SLO{
				{
					Name:        "requests-availability",
					Objective:   99.9,
					Description: "blah blah",
					SLI: slothv1.SLI{Events: &slothv1.SLIEvents{
						ErrorQuery: "sum(rate(mattermost_api_time_count{job='mm-" + clusterInstallation.Namespace[0:4] + "',code=~'(5..|429)'}[{{.window}}]))",
						TotalQuery: "sum(rate(mattermost_api_time_count{job='mm-" + clusterInstallation.Namespace[0:4] + "'}[{{.window}}]))",
					}},
					Alerting: slothv1.Alerting{
						Name: "mm-" + clusterInstallation.Namespace[0:4] + "-service-HighAPIErrorRate",
						Labels: map[string]string{
							"category": "availability",
						},
						Annotations: map[string]string{
							"summary": "High error rate on requests responses",
						},
					},
				}},
		},
	}
	ctx := context.TODO()
	_, err := k8sClient.SlothClientsetV1.SlothV1().PrometheusServiceLevels("prometheus").Get(ctx, sli.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		k8sClient.SlothClientsetV1.SlothV1().PrometheusServiceLevels("prometheus").Create(ctx, sli, metav1.CreateOptions{})
		return nil
	}

	return err
}

func (provisioner *KopsProvisioner) deleteInstallationSLI(clusterInstallation *model.ClusterInstallation, k8sClient *k8s.KubeClient) error {

	sli := clusterInstallation.InstallationID

	ctx := context.TODO()
	_, err := k8sClient.SlothClientsetV1.SlothV1().PrometheusServiceLevels("prometheus").Get(ctx, sli, metav1.GetOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		return nil
	}
	err = k8sClient.SlothClientsetV1.SlothV1().PrometheusServiceLevels("prometheus").Delete(ctx, sli, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to delete cluster installation sli")
	}
	return nil
}
