// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (provisioner *KopsProvisioner) makeSLIs(clusterInstallation *model.ClusterInstallation) slothv1.PrometheusServiceLevel {
	installationName := makeClusterInstallationName(clusterInstallation)
	sli := slothv1.PrometheusServiceLevel{
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
					Objective:   99.5,
					Description: "Availability metric for mattermost API",
					SLI: slothv1.SLI{Events: &slothv1.SLIEvents{
						ErrorQuery: "sum(rate(mattermost_api_time_count{job='" + installationName + "',status_code=~'(5..|429|499)'}[{{.window}}]))",
						TotalQuery: "sum(rate(mattermost_api_time_count{job='" + installationName + "'}[{{.window}}]))",
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

func (provisioner *KopsProvisioner) createInstallationSLI(clusterInstallation *model.ClusterInstallation, k8sClient *k8s.KubeClient, logger log.FieldLogger) error {
	wait := 60
	sli := provisioner.makeSLIs(clusterInstallation)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
	defer cancel()
	_, err := k8sClient.SlothClientsetV1.SlothV1().PrometheusServiceLevels("prometheus").Create(ctx, &sli, metav1.CreateOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		logger.Debugf("Sloth CRD doesn't exist on cluster: %s", err)
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "failed to create cluster installation sli")
	}
	return nil
}

func (provisioner *KopsProvisioner) updateInstallationSLI(sli slothv1.PrometheusServiceLevel, k8sClient *k8s.KubeClient, logger log.FieldLogger) error {
	wait := 60
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
	defer cancel()
	obj, err := k8sClient.SlothClientsetV1.SlothV1().PrometheusServiceLevels("prometheus").Get(ctx, sli.Name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get cluster installation sli")
	}
	sli.ResourceVersion = obj.GetResourceVersion()
	_, err = k8sClient.SlothClientsetV1.SlothV1().PrometheusServiceLevels("prometheus").Update(ctx, &sli, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update cluster installation sli")
	}
	return nil
}

func (provisioner *KopsProvisioner) createOrUpdateInstallationSLI(clusterInstallation *model.ClusterInstallation, k8sClient *k8s.KubeClient, logger log.FieldLogger) error {
	wait := 60
	sli := provisioner.makeSLIs(clusterInstallation)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
	defer cancel()
	_, err := k8sClient.SlothClientsetV1.SlothV1().PrometheusServiceLevels("prometheus").Get(ctx, sli.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get cluster installation sli")
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		err = provisioner.createInstallationSLI(clusterInstallation, k8sClient, logger)
		if err != nil {
			return errors.Wrap(err, "failed to create cluster installation sli")
		}
		return nil
	}

	err = provisioner.updateInstallationSLI(sli, k8sClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to update cluster installation sli")
	}
	return nil
}

func (provisioner *KopsProvisioner) deleteInstallationSLI(clusterInstallation *model.ClusterInstallation, k8sClient *k8s.KubeClient, logger log.FieldLogger) error {
	wait := 60
	sli := clusterInstallation.InstallationID
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
	defer cancel()
	_, err := k8sClient.SlothClientsetV1.SlothV1().PrometheusServiceLevels("prometheus").Get(ctx, sli, metav1.GetOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		logger.Debugf("Sloth CRD doesn't exist on cluster: %s", err)
		return nil
	}
	err = k8sClient.SlothClientsetV1.SlothV1().PrometheusServiceLevels("prometheus").Delete(ctx, sli, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to delete cluster installation sli")
	}
	return nil
}
