// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package k8s

import (
	log "github.com/sirupsen/logrus"

	mmclientv1alpha1 "github.com/mattermost/mattermost-operator/pkg/client/clientset/versioned"
	mmclientv1beta1 "github.com/mattermost/mattermost-operator/pkg/client/v1beta1/clientset/versioned"
	apixclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	kubeagclient "k8s.io/kube-aggregator/pkg/client/clientset_generated/clientset"
)

// KubeClient interfaces with a Kubernetes cluster in the same way kubectl would.
type KubeClient struct {
	config                     *rest.Config
	Clientset                  kubernetes.Interface
	ApixClientset              apixclient.Interface
	MattermostClientsetV1Alpha mmclientv1alpha1.Interface
	MattermostClientsetV1Beta  mmclientv1beta1.Interface
	KubeagClientSet            kubeagclient.Interface
	logger                     log.FieldLogger
}

// NewFromConfig takes in an already created Kubernetes config object, and returns a KubeClient for accessing the kubernetes API
func NewFromConfig(config *rest.Config, logger log.FieldLogger) (*KubeClient, error) {
	return createKubeClient(config, logger)
}

// NewFromFile returns a new KubeClient for accessing the kubernetes API. (previously named 'New')
func NewFromFile(configLocation string, logger log.FieldLogger) (*KubeClient, error) {
	config, err := clientcmd.BuildConfigFromFlags("", configLocation)
	if err != nil {
		return nil, err
	}

	return createKubeClient(config, logger)
}

func createKubeClient(config *rest.Config, logger log.FieldLogger) (*KubeClient, error) {
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	apixClientset, err := apixclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	mattermostV1AlphaClientset, err := mmclientv1alpha1.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	mattermostV1BetaClientset, err := mmclientv1beta1.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	kubeagClientset, err := kubeagclient.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return &KubeClient{
			config:                     config,
			Clientset:                  clientset,
			MattermostClientsetV1Alpha: mattermostV1AlphaClientset,
			MattermostClientsetV1Beta:  mattermostV1BetaClientset,
			ApixClientset:              apixClientset,
			KubeagClientSet:            kubeagClientset,
			logger:                     logger,
		},
		nil
}

func (kc *KubeClient) getKubeConfigClientset() (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(kc.config)
}

// GetConfig exposes the rest.Config for use with other k8s packages.
func (kc *KubeClient) GetConfig() *rest.Config {
	return kc.config
}
