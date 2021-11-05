// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	monitoringV1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	mmv1alpha1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"
	apixv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apixv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (kc *KubeClient) createOrUpdateCustomResourceDefinitionBetaV1(crd *apixv1beta1.CustomResourceDefinition) (metav1.Object, error) {
	ctx := context.TODO()
	oldCRD, err := kc.ApixClientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(ctx, crd.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.ApixClientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(ctx, crd, metav1.CreateOptions{})
	}

	crd.ResourceVersion = oldCRD.ResourceVersion
	return kc.ApixClientset.ApiextensionsV1beta1().CustomResourceDefinitions().Update(ctx, crd, metav1.UpdateOptions{})
}

func (kc *KubeClient) createOrUpdateCustomResourceDefinitionV1(crd *apixv1.CustomResourceDefinition) (metav1.Object, error) {
	ctx := context.TODO()
	oldCRD, err := kc.ApixClientset.ApiextensionsV1().CustomResourceDefinitions().Get(ctx, crd.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.ApixClientset.ApiextensionsV1().CustomResourceDefinitions().Create(ctx, crd, metav1.CreateOptions{})
	}

	crd.ResourceVersion = oldCRD.ResourceVersion
	return kc.ApixClientset.ApiextensionsV1().CustomResourceDefinitions().Update(ctx, crd, metav1.UpdateOptions{})
}

func (kc *KubeClient) createOrUpdateClusterInstallation(namespace string, ci *mmv1alpha1.ClusterInstallation) (metav1.Object, error) {
	ctx := context.TODO()
	_, err := kc.MattermostClientsetV1Alpha.MattermostV1alpha1().ClusterInstallations(namespace).Get(ctx, ci.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.MattermostClientsetV1Alpha.MattermostV1alpha1().ClusterInstallations(namespace).Create(ctx, ci, metav1.CreateOptions{})
	}

	return kc.MattermostClientsetV1Alpha.MattermostV1alpha1().ClusterInstallations(namespace).Update(ctx, ci, metav1.UpdateOptions{})
}

func (kc *KubeClient) createOrUpdatePodMonitor(namespace string, pm *monitoringV1.PodMonitor) (metav1.Object, error) {
	wait := 60
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
	defer cancel()
	_, err := kc.MonitoringClientsetV1.MonitoringV1().PodMonitors(namespace).Get(ctx, pm.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.MonitoringClientsetV1.MonitoringV1().PodMonitors(namespace).Create(ctx, pm, metav1.CreateOptions{})
	}

	patch, err := json.Marshal(pm)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal new Pod Monitor")
	}

	return kc.MonitoringClientsetV1.MonitoringV1().PodMonitors(namespace).Patch(ctx, pm.GetName(), types.MergePatchType, patch, metav1.PatchOptions{})
}
