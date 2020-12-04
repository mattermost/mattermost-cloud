// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"
	"io/ioutil"
	"path"
	"regexp"
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
		return fmt.Errorf("terraform cluster_name (%s) does not match kops_name from provided ID (%s)", out, kopsName)
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
			if strings.HasSuffix(service.Name, "internal") || strings.HasSuffix(service.Name, "query") {
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

// GetPublicLoadBalancerEndpoint returns the public load balancer endpoint of the NGINX service.
func (provisioner *KopsProvisioner) GetPublicLoadBalancerEndpoint(cluster *model.Cluster, namespace string) (string, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":         cluster.ID,
		"nginx-namespace": namespace,
	})
	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return "", errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	err = kops.ExportKubecfg(cluster.ProvisionerMetadataKops.Name)
	if err != nil {
		return "", errors.Wrap(err, "failed to export kubecfg")
	}

	k8sClient, err := k8s.NewFromFile(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return "", err
	}

	ctx := context.TODO()
	services, err := k8sClient.Clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, service := range services.Items {
		if !strings.HasSuffix(service.Name, "internal") {
			if service.Status.LoadBalancer.Ingress != nil {
				endpoint := service.Status.LoadBalancer.Ingress[0].Hostname
				if endpoint == "" {
					return "", errors.New("loadbalancer endpoint value is empty")
				}

				return endpoint, nil
			}
		}
	}
	return "", errors.New("failed to get NGINX load balancer endpoint")
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

			igManifest, err := kops.GetInstanceGroupYAML(kopsMetadata.Name, ig.Metadata.Name)
			if err != nil {
				return errors.Wrap(err, "failed to get YAML output for instance group")
			}
			igManifest, err = grossKopsReplaceImage(igManifest, ami)
			if err != nil {
				return errors.Wrap(err, "failed to replace image value in YAML")
			}

			igFilename := fmt.Sprintf("%s-ig.yaml", ig.Metadata.Name)
			err = ioutil.WriteFile(path.Join(kops.GetTempDir(), igFilename), []byte(igManifest), 0600)
			if err != nil {
				return errors.Wrap(err, "failed to write new YAML file")
			}
			_, err = kops.Replace(igFilename)
			if err != nil {
				return errors.Wrap(err, "failed to update instance group")
			}
		}
	}

	return nil
}

// grossKopsReplaceSize is a manual find-and-replace flow for updating a raw
// kops instance group YAML manifest with new sizing values.
// TODO: remove once new `kops set instancegroup` functionality is available.
//
// Example Manifest:
//
// apiVersion: kops.k8s.io/v1alpha2
// kind: InstanceGroup
// spec:
//   machineType: m5.large
//   maxSize: 2
//   minSize: 2
func grossKopsReplaceSize(input, machineType, min, max string) (string, error) {
	if len(machineType) != 0 {
		machineTypeRE := regexp.MustCompile(`  machineType: .*\n`)
		machineTypeMatches := len(machineTypeRE.FindAllStringIndex(input, -1))
		if machineTypeMatches != 1 {
			return "", errors.Errorf("expected to find one machineType match, but found %d", machineTypeMatches)
		}
		input = machineTypeRE.ReplaceAllString(input, fmt.Sprintf("  machineType: %s\n", machineType))
	}

	if len(min) != 0 && min != "0" {
		minRE := regexp.MustCompile(`  minSize: ?\d+\n`)
		minMatches := len(minRE.FindAllStringIndex(input, -1))
		if minMatches != 1 {
			return "", errors.Errorf("expected to find one minSize match, but found %d", minMatches)
		}
		input = minRE.ReplaceAllString(input, fmt.Sprintf("  minSize: %s\n", min))
	}

	if len(max) != 0 && max != "0" {
		maxRE := regexp.MustCompile(`  maxSize: ?\d+\n`)
		maxMatches := len(maxRE.FindAllStringIndex(input, -1))
		if maxMatches != 1 {
			return "", errors.Errorf("expected to find one maxSize match, but found %d", maxMatches)
		}
		input = maxRE.ReplaceAllString(input, fmt.Sprintf("  maxSize: %s\n", max))
	}

	return input, nil
}

// grossKopsReplaceImage is a manual find-and-replace flow for updating a raw
// kops instance group YAML manifest with a new image value.
// TODO: remove once new `kops set instancegroup` functionality is available.
//
// Example Manifest:
//
// apiVersion: kops.k8s.io/v1alpha2
// kind: InstanceGroup
// spec:
//   image: kope.io/k8s-1.15-debian-stretch-amd64-hvm-ebs-2020-01-17
func grossKopsReplaceImage(input, image string) (string, error) {
	imageRE := regexp.MustCompile(`  image: .*\n`)
	imageMatches := len(imageRE.FindAllStringIndex(input, -1))
	if imageMatches != 1 {
		return "", errors.Errorf("expected to find one image match, but found %d", imageMatches)
	}
	input = imageRE.ReplaceAllString(input, fmt.Sprintf("  image: %s\n", image))

	return input, nil
}
