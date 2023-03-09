// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package prometheus

import (
	"context"
	"time"

	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	slothv1 "github.com/slok/sloth/pkg/kubernetes/api/sloth/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	Namespace            = "prometheus"
	SlothDefaultWaitTime = time.Second * 60
)

func createPrometheusServiceLevel(psl slothv1.PrometheusServiceLevel, k8sClient *k8s.KubeClient, logger log.FieldLogger) error {
	ctx, cancel := context.WithTimeout(context.Background(), SlothDefaultWaitTime)
	defer cancel()
	_, err := k8sClient.SlothClientsetV1.SlothV1().PrometheusServiceLevels(Namespace).Create(ctx, &psl, metav1.CreateOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		logger.Debugf("Sloth CRD doesn't exist on cluster: %s", err)
		return nil
	}
	if err != nil {
		return errors.Wrap(err, "failed to create cluster installation sli")
	}
	return nil
}

func updatePrometheusServiceLevel(psl slothv1.PrometheusServiceLevel, k8sClient *k8s.KubeClient, logger log.FieldLogger) error {
	ctx, cancel := context.WithTimeout(context.Background(), SlothDefaultWaitTime)
	defer cancel()
	obj, err := k8sClient.SlothClientsetV1.SlothV1().PrometheusServiceLevels(Namespace).Get(ctx, psl.Name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get cluster installation sli")
	}
	psl.ResourceVersion = obj.GetResourceVersion()
	_, err = k8sClient.SlothClientsetV1.SlothV1().PrometheusServiceLevels(Namespace).Update(ctx, &psl, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to update cluster installation sli")
	}
	return nil
}

func CreateOrUpdateClusterPrometheusServiceLevel(psl slothv1.PrometheusServiceLevel, k8sClient *k8s.KubeClient, logger log.FieldLogger) error {
	ctx, cancel := context.WithTimeout(context.Background(), SlothDefaultWaitTime)
	defer cancel()
	_, err := k8sClient.SlothClientsetV1.SlothV1().PrometheusServiceLevels(Namespace).Get(ctx, psl.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return errors.Wrap(err, "failed to get cluster installation sli")
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		err = createPrometheusServiceLevel(psl, k8sClient, logger)
		if err != nil {
			return errors.Wrap(err, "failed to create cluster installation sli")
		}
		return nil
	}

	err = updatePrometheusServiceLevel(psl, k8sClient, logger)
	if err != nil {
		return errors.Wrap(err, "failed to update cluster installation sli")
	}
	return nil
}
