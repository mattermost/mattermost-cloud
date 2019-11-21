package provisioner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/k8s"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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

// waitForHelmRunning is used to check when Helm is ready to install charts.
func waitForHelmRunning(ctx context.Context, configPath string) error {
	for {
		cmd := exec.Command("helm", "ls", "--kubeconfig", configPath)
		var out bytes.Buffer
		cmd.Stderr = &out
		cmd.Run()
		if out.String() == "" {
			return nil
		}
		select {
		case <-ctx.Done():
			return errors.Wrap(ctx.Err(), "timed out waiting for helm to become ready")
		case <-time.After(5 * time.Second):
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

// addHelmRepo adds a new Helm repo
func addHelmRepo(repoName, repoURL string, logger log.FieldLogger) error {
	var cmd *exec.Cmd
	arguments := []string{
		repoName,
		repoURL,
	}

	cmd = exec.Command("helm repo add", arguments...)

	logger.WithFields(log.Fields{
		"cmd":  cmd.Path,
		"args": cmd.Args,
	}).Info("Invoking command")

	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// helmRepoUpdate updates the helm repos to get latest available charts
func helmRepoUpdate(logger log.FieldLogger) error {
	var cmd *exec.Cmd
	cmd = exec.Command("helm repo update")

	logger.WithFields(log.Fields{
		"cmd": cmd.Path,
	}).Info("Invoking command")

	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// installHelmChart is used to install Helm charts.
func installHelmChart(chart helmDeployment, configPath string, logger log.FieldLogger) error {
	var cmd *exec.Cmd
	arguments := []string{
		"--debug",
		"install",
		"--kubeconfig", configPath,
		"-f", chart.valuesPath,
		chart.chartName,
		"--namespace", chart.namespace,
		"--name", chart.chartDeploymentName}

	if chart.setArgument != "" {
		arguments = append(arguments, "--set", chart.setArgument)
	}

	cmd = exec.Command("helm", arguments...)

	logger.WithFields(log.Fields{
		"cmd":  cmd.Path,
		"args": cmd.Args,
	}).Info("Invoking command")

	err := cmd.Run()
	if err != nil {
		return err
	}
	return nil
}

// helmSetup is used for the initial setup of Helm in cluster.
func helmSetup(logger log.FieldLogger, kops *kops.Cmd) error {
	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return err
	}

	logger.Info("Initializing Helm in the cluster")
	err = exec.Command("helm", "--debug", "--kubeconfig", kops.GetKubeConfigPath(), "init", "--upgrade").Run()
	if err != nil {
		return err
	}

	logger.Info("Creating Tiller service account")
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "tiller"},
	}
	_, err = k8sClient.Clientset.CoreV1().ServiceAccounts("kube-system").Create(serviceAccount)
	if err != nil {
		return err
	}
	logger.Info("Successfully created Tiller service account")

	logger.Info("Creating Tiller cluster role bind")
	roleBinding := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{Name: "tiller-cluster-rule"},
		Subjects: []rbacv1.Subject{
			{Kind: "ServiceAccount", Name: "tiller", Namespace: "kube-system"},
		},
		RoleRef: rbacv1.RoleRef{Kind: "ClusterRole", Name: "cluster-admin"},
	}

	_, err = k8sClient.Clientset.RbacV1().ClusterRoleBindings().Create(roleBinding)
	if err != nil {
		return err
	}
	logger.Info("Successfully created cluster role bind")

	logger.Info("Upgrade Helm")

	cmd := exec.Command("helm", "--debug", "--kubeconfig", kops.GetKubeConfigPath(), "init", "--service-account", "tiller", "--upgrade")

	logger.WithFields(log.Fields{
		"cmd":  cmd.Path,
		"args": cmd.Args,
	}).Info("Invoking command")

	err = cmd.Run()
	if err != nil {
		return err
	}
	return nil
}
