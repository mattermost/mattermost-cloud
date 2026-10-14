// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// ClusterInstallationProvisioner function returns an implementation of ClusterInstallationProvisioner interface
// based on specified Custom Resource version.
func (provisioner Provisioner) ClusterInstallationProvisioner(version string) supervisor.ClusterInstallationProvisioner {
	if version != model.V1betaCRVersion {
		provisioner.logger.Errorf("Unexpected resource version: %s", version)
	}

	return provisioner
}

// GetClusterResources returns a snapshot of resources of a given cluster.
func (provisioner Provisioner) GetClusterResources(cluster *model.Cluster, onlySchedulable bool, logger log.FieldLogger) (*k8s.ClusterResources, error) {
	logger = logger.WithField("cluster", cluster.ID)

	configLocation, err := provisioner.getClusterKubecfg(cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kube config path")
	}
	return getClusterResources(configLocation, onlySchedulable, logger)
}

// GetPublicLoadBalancerEndpoint returns the public load balancer endpoint of the NGINX service.
func (provisioner Provisioner) GetPublicLoadBalancerEndpoint(cluster *model.Cluster, namespace string) (string, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":         cluster.ID,
		"nginx-namespace": namespace,
	})

	configLocation, err := provisioner.getClusterKubecfg(cluster)
	if err != nil {
		return "", errors.Wrap(err, "failed to get kube config path")
	}

	return getPublicLoadBalancerEndpoint(configLocation, namespace, logger)
}

// GetGatewayLoadBalancerEndpoint returns the public load balancer hostname published
// in the status of the given Gateway.
func (provisioner Provisioner) GetGatewayLoadBalancerEndpoint(cluster *model.Cluster, namespace, name string) (string, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":           cluster.ID,
		"gateway-namespace": namespace,
		"gateway":           name,
	})

	configLocation, err := provisioner.getClusterKubecfg(cluster)
	if err != nil {
		return "", errors.Wrap(err, "failed to get kube config path")
	}

	k8sClient, err := k8s.NewFromFile(configLocation, logger)
	if err != nil {
		return "", errors.Wrap(err, "failed to create k8s client from file")
	}

	return k8sClient.GetGatewayHostname(namespace, name)
}
