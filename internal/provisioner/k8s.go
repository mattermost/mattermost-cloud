// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
)

type kube interface {
	getKubeClient(cluster *model.Cluster) (*k8s.KubeClient, error)
	getKubeConfigPath(cluster *model.Cluster) (string, error)
}

func (provisioner Provisioner) getKubeOption(provisionerOption string) kube {
	if provisionerOption == model.ProvisionerEKS {
		return provisioner.eksProvisioner
	}

	return provisioner.kopsProvisioner
}

func (provisioner Provisioner) k8sClient(cluster *model.Cluster) (*k8s.KubeClient, error) {
	return provisioner.getKubeOption(cluster.Provisioner).getKubeClient(cluster)
}

func (provisioner Provisioner) getClusterKubecfg(cluster *model.Cluster) (string, error) {
	return provisioner.getKubeOption(cluster.Provisioner).getKubeConfigPath(cluster)
}
