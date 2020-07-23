// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apiregistrationv1beta1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1beta1"
)

func (kc *KubeClient) createOrUpdateAPIServer(apiRegistration *apiregistrationv1beta1.APIService) (metav1.Object, error) {
	_, err := kc.KubeagClientSet.ApiregistrationV1beta1().APIServices().Get(apiRegistration.GetName(), metav1.GetOptions{})
	if err != nil && !k8sErrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8sErrors.IsNotFound(err) {
		return kc.KubeagClientSet.ApiregistrationV1beta1().APIServices().Create(apiRegistration)
	}

	return kc.KubeagClientSet.ApiregistrationV1beta1().APIServices().Update(apiRegistration)
}
