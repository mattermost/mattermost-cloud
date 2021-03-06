// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	"github.com/sirupsen/logrus"

	mmfake "github.com/mattermost/mattermost-operator/pkg/client/clientset/versioned/fake"
	mmfakebeta "github.com/mattermost/mattermost-operator/pkg/client/v1beta1/clientset/versioned/fake"
	apixfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	kubeagfake "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset/fake"
)

func newTestKubeClient() *KubeClient {
	return &KubeClient{
		config:                     &rest.Config{},
		Clientset:                  fake.NewSimpleClientset(),
		ApixClientset:              apixfake.NewSimpleClientset(),
		MattermostClientsetV1Alpha: mmfake.NewSimpleClientset(),
		MattermostClientsetV1Beta:  mmfakebeta.NewSimpleClientset(),
		KubeagClientSet:            kubeagfake.NewSimpleClientset(),
		logger:                     logrus.New(),
	}
}
