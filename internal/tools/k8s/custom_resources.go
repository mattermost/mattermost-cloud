// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"encoding/json"

	"github.com/pkg/errors"

	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
	apixv1beta1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (kc *KubeClient) createOrUpdateCustomResourceDefinition(crd *apixv1beta1.CustomResourceDefinition) (metav1.Object, error) {
	_, err := kc.ApixClientset.ApiextensionsV1beta1().CustomResourceDefinitions().Get(crd.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.ApixClientset.ApiextensionsV1beta1().CustomResourceDefinitions().Create(crd)
	}

	// TODO: investigate issue where standard update fails
	// Trying to use a standard update on CRDs failed for the mysql operator
	// custom resources definitions. This seems to be related to an issue
	// where a last-modified value is set after the CRD is deployed to the
	// kubernetes cluster.
	// Error: invalid: metadata.resourceVersion: Invalid value: 0x0: must be
	// specified for an update
	// Workaround: https://github.com/zalando/postgres-operator/pull/558
	patch, err := json.Marshal(crd)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal new Custom Resource Defintion")
	}

	return kc.ApixClientset.ApiextensionsV1beta1().CustomResourceDefinitions().Patch(crd.Name, types.MergePatchType, patch)
}

func (kc *KubeClient) createOrUpdateClusterInstallation(namespace string, ci *mmv1alpha1.ClusterInstallation) (metav1.Object, error) {
	_, err := kc.MattermostClientset.MattermostV1alpha1().ClusterInstallations(namespace).Get(ci.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.MattermostClientset.MattermostV1alpha1().ClusterInstallations(namespace).Create(ci)
	}

	return kc.MattermostClientset.MattermostV1alpha1().ClusterInstallations(namespace).Update(ci)
}
