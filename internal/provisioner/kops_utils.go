package provisioner

import (
	"context"
	"fmt"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/k8s"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/internal/tools/terraform"
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
				_, err := k8sClient.Clientset.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
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

// getLoadBalancerEndpoint is used to get the endpoint of the internal ingress.
func getLoadBalancerEndpoint(ctx context.Context, namespace string, logger log.FieldLogger, configPath string) (string, error) {
	k8sClient, err := k8s.New(configPath, logger)
	if err != nil {
		return "", err
	}
	for {
		services, err := k8sClient.Clientset.CoreV1().Services(namespace).List(metav1.ListOptions{})
		if err != nil {
			return "", err
		}
		for _, service := range services.Items {
			if service.Status.LoadBalancer.Ingress != nil {
				endpoint := service.Status.LoadBalancer.Ingress[0].Hostname
				if endpoint == "" {
					return "", errors.New("loadbalancer endpoint value is empty")
				}

				return endpoint, nil
			}
		}

		select {
		case <-ctx.Done():
			return "", errors.Wrap(ctx.Err(), "timed out waiting for helm to become ready")
		case <-time.After(5 * time.Second):
		}
	}
}

// GetNGINXLoadBalancerEndpoint returns the load balancer endpoint of the NGINX service.
func (provisioner *KopsProvisioner) GetNGINXLoadBalancerEndpoint(cluster *model.Cluster, namespace string) (string, error) {
	logger := provisioner.logger.WithFields(log.Fields{
		"cluster":         cluster.ID,
		"nginx-namespace": namespace,
	})
	kops, err := kops.New(provisioner.s3StateStore, logger)
	if err != nil {
		return "", errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	kopsMetadata, err := model.NewKopsMetadata(cluster.ProvisionerMetadata)
	if err != nil {
		return "", errors.Wrap(err, "failed to parse provisioner metadata")
	}

	err = kops.ExportKubecfg(kopsMetadata.Name)
	if err != nil {
		return "", errors.Wrap(err, "failed to export kubecfg")
	}

	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return "", err
	}

	services, err := k8sClient.Clientset.CoreV1().Services(namespace).List(metav1.ListOptions{})
	if err != nil {
		return "", err
	}
	for _, service := range services.Items {
		if service.Status.LoadBalancer.Ingress != nil {
			endpoint := service.Status.LoadBalancer.Ingress[0].Hostname
			if endpoint == "" {
				return "", errors.New("loadbalancer endpoint value is empty")
			}

			return endpoint, nil
		}
	}
	return "", errors.New("failed to get NGINX load balancer endpoint")
}
