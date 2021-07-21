// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//+build e2e

package pkg

import (
	"path/filepath"

	restclient "k8s.io/client-go/rest"

	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

// GetK8sConfig gets K8s config either from inside the cluster or local directory.
func GetK8sConfig() (*restclient.Config, error) {
	k8sConfig, err := restclient.InClusterConfig()
	if err != nil {
		home := homedir.HomeDir()
		k8sConfPath := filepath.Join(home, ".kube", "config")
		k8sConfig, err = clientcmd.BuildConfigFromFlags("", k8sConfPath)
		if err != nil {
			return nil, err
		}
	}

	return k8sConfig, nil
}
