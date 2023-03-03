// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/internal/tools/terraform"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// verifyTerraformAndKopsMatch looks at terraform output and verifies that the
// given kops name matches. This should only catch errors where terraform output
// was incorrectly created from kops or if the terraform client is targeting the
// wrong directory, but should be used as a final sanity check before invoking
// terraform commands.
func verifyTerraformAndKopsMatch(kopsName string, terraformClient *terraform.Cmd, logger log.FieldLogger) error {
	out, ok, err := terraformClient.Output("cluster_name")
	if err != nil {
		return err
	}
	if !ok {
		logger.Warn("No cluster_name in terraform config, skipping check")
		return nil
	}
	if out != kopsName {
		return errors.Errorf("terraform cluster_name (%s) does not match kops_name from provided ID (%s)", out, kopsName)
	}

	return nil
}

// Override the version to make match the nil value in the custom resource.
// TODO: this could probably be better. We may want the operator to understand
// default values instead of needing to pass in empty values.
func translateMattermostVersion(version string) string {
	if version == "stable" {
		return ""
	}

	return version
}

func makeClusterInstallationName(clusterInstallation *model.ClusterInstallation) string {
	// TODO: Once https://mattermost.atlassian.net/browse/MM-15467 is fixed, we can use the
	// full namespace as part of the name. For now, truncate to keep within the existing limit
	// of 60 characters.
	return fmt.Sprintf("mm-%s", clusterInstallation.Namespace[0:4])
}

// waitForNamespacesDeleted is used to check when all of the provided namespaces
// have been fully terminated.
func waitForNamespacesDeleted(ctx context.Context, namespaces []string, k8sClient *k8s.KubeClient) error {
	for {
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "timed out waiting for namespaces to become fully terminated")
		default:
			var shouldWait bool
			for _, namespace := range namespaces {
				_, err := k8sClient.Clientset.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
				if err != nil && k8sErrors.IsNotFound(err) {
					continue
				}

				shouldWait = true
				break
			}

			if !shouldWait {
				return nil
			}

			time.Sleep(5 * time.Second)
		}
	}
}

// getPrivateLoadBalancerEndpoint returns the private load balancer endpoint of the NGINX service.
func getPrivateLoadBalancerEndpoint(ctx context.Context, namespace string, logger log.FieldLogger, configPath string) (string, error) {
	k8sClient, err := k8s.NewFromFile(configPath, logger)
	if err != nil {
		return "", err
	}

	for {
		services, err := k8sClient.Clientset.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			return "", err
		}
		for _, service := range services.Items {
			if service.Spec.Type == "LoadBalancer" || strings.HasSuffix(service.Name, "query") {
				if service.Status.LoadBalancer.Ingress != nil {
					endpoint := service.Status.LoadBalancer.Ingress[0].Hostname
					if endpoint == "" {
						return "", errors.New("loadbalancer endpoint value is empty")
					}

					return endpoint, nil
				}
			}
		}
		select {
		case <-ctx.Done():
			return "", errors.Wrap(ctx.Err(), "timed out waiting for internal load balancer to become ready")
		case <-time.After(5 * time.Second):
		}
	}
}

// getElasticLoadBalancerInfo returns the private load balancer endpoint and type of the NGINX service.
func getElasticLoadBalancerInfo(namespace string, logger log.FieldLogger, configPath string) (string, string, error) {
	k8sClient, err := k8s.NewFromFile(configPath, logger)
	if err != nil {
		return "", "", err
	}

	services, err := k8sClient.Clientset.CoreV1().Services(namespace).List(context.TODO(), metav1.ListOptions{})
	if err != nil {
		return "", "", err
	}

	for _, service := range services.Items {
		if service.Spec.Type == "LoadBalancer" {
			if service.Status.LoadBalancer.Ingress != nil {
				endpoint := service.Status.LoadBalancer.Ingress[0].Hostname
				if endpoint != "" {
					return endpoint, service.Annotations["service.beta.kubernetes.io/aws-load-balancer-type"], nil
				}
			}
		}
	}

	return "", "", nil
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

func updateKopsInstanceGroupAMIs(kops *kops.Cmd, kopsMetadata *model.KopsMetadata, logger log.FieldLogger) error {
	if len(kopsMetadata.ChangeRequest.AMI) == 0 {
		logger.Info("Skipping cluster AMI update")
		return nil
	}

	instanceGroups, err := kops.GetInstanceGroupsJSON(kopsMetadata.Name)
	if err != nil {
		return errors.Wrap(err, "failed to get instance groups")
	}

	var ami string
	for _, ig := range instanceGroups {
		if ig.Spec.Image != kopsMetadata.ChangeRequest.AMI {
			if kopsMetadata.ChangeRequest.AMI == "latest" {
				// Setting the image value to "" leads kops to autoreplace it with
				// the default image for that kubernetes release.
				logger.Infof("Updating instance group '%s' image value the default kops image", ig.Metadata.Name)
				ami = ""
			} else {
				logger.Infof("Updating instance group '%s' image value to '%s'", ig.Metadata.Name, kopsMetadata.ChangeRequest.AMI)
				ami = kopsMetadata.ChangeRequest.AMI
			}

			err = kops.SetInstanceGroup(kopsMetadata.Name, ig.Metadata.Name, fmt.Sprintf("spec.image=%s", ami))
			if err != nil {
				return errors.Wrap(err, "failed to update instance group ami")
			}
		}
	}

	return nil
}

func updateKopsInstanceGroupValue(kops *kops.Cmd, kopsMetadata *model.KopsMetadata, value string) error {

	instanceGroups, err := kops.GetInstanceGroupsJSON(kopsMetadata.Name)
	if err != nil {
		return errors.Wrap(err, "failed to get instance groups")
	}

	for _, ig := range instanceGroups {

		err = kops.SetInstanceGroup(kopsMetadata.Name, ig.Metadata.Name, value)
		if err != nil {
			return errors.Wrapf(err, "failed to update value %s", value)
		}
	}

	return nil
}
