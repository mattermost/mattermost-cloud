package provisioner

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mattermost/mattermost-cloud/internal/tools/k8s"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
)

// helmDeployment deploys Helm charts.
type helmDeployment struct {
	chartDeploymentName string
	chartName           string
	namespace           string
	setArgument         string
	valuesPath          string
	version             string

	cluster         *model.Cluster
	kopsProvisioner *KopsProvisioner
	kops            *kops.Cmd
	logger          log.FieldLogger
}

func installHelm(kops *kops.Cmd, logger log.FieldLogger) error {
	logger.Info("Installing Helm")

	err := helmSetup(logger, kops)
	if err != nil {
		return errors.Wrap(err, "couldn't install Helm")
	}

	wait := 120
	logger.Infof("Waiting up to %d seconds for helm to become ready...", wait)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
	defer cancel()
	err = waitForHelmRunning(ctx, kops.GetKubeConfigPath())
	if err != nil {
		return errors.Wrap(err, "Helm didn't start as expected, or we couldn't detect it")
	}

	logger.Info("Updating all Helm repos.")
	return helmRepoUpdate(logger)
}

func (d *helmDeployment) Update() error {
	logger := d.logger.WithField("helm-update", d.chartName)

	logger.Infof("Refreshing helm chart %s -- may trigger service upgrade", d.chartName)
	err := upgradeHelmChart(*d, d.kops.GetKubeConfigPath(), logger)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("got an error trying to upgrade the helm chart %s", d.chartName))
	}
	return nil
}

// Create will install a given HelmDeployment in the cluster specified in that object.
func (d *helmDeployment) Create() error {
	logger := d.logger.WithField("helm-create", d.chartName)
	logger.Infof("Installing helm chart %s", d.chartName)
	return installHelmChart(*d, d.kops.GetKubeConfigPath(), logger)
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

// helmRepoUpdate updates the helm repos to get latest available charts
func helmRepoUpdate(logger log.FieldLogger) error {
	var cmd *exec.Cmd
	arguments := []string{
		"repo",
		"update",
	}

	cmd = exec.Command("helm", arguments...)

	logger.WithFields(log.Fields{
		"cmd": cmd.Path,
	}).Info("Invoking command")

	output, err := cmd.Output()
	if err != nil {
		if len(output) > 0 {
			log.Debugf("helm output was:\n%s\n", string(output))
		}
		return errors.Wrap(err, "Failed to execute Helm to update the Helm repos")
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

	if chart.version != "" {
		arguments = append(arguments, "--version", chart.version)
	}

	cmd = exec.Command("helm", arguments...)

	logger.WithFields(log.Fields{
		"cmd":  cmd.Path,
		"args": cmd.Args,
	}).Info("Invoking command")

	out, err := cmd.Output()
	if err != nil {
		logger.Debugf("Helm output was %s", string(out))
		return errors.Wrap(err, fmt.Sprintf("Couldn't execute Helm when trying to install the chart %s", chart.chartName))
	}
	return nil
}

// upgradeHelmChart is used to upgrade Helm deployments.
func upgradeHelmChart(chart helmDeployment, configPath string, logger log.FieldLogger) error {
	var cmd *exec.Cmd
	arguments := []string{
		"--debug",
		"upgrade",
		chart.chartDeploymentName,
		chart.chartName,
		"--kubeconfig", configPath,
		"-f", chart.valuesPath,
		"--namespace", chart.namespace,
	}

	if chart.setArgument != "" {
		arguments = append(arguments, "--set", chart.setArgument)
	}

	if chart.version != "" {
		arguments = append(arguments, "--version", chart.version)
	}

	cmd = exec.Command("helm", arguments...)

	logger.WithFields(log.Fields{
		"cmd":  cmd.Path,
		"args": cmd.Args,
	}).Info("Invoking command")

	output, err := cmd.Output()
	if len(output) > 0 {
		log.Debugf("Helm output was:\n%s\n", string(output))
	}

	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("failed to upgrade Helm chart %s", chart.chartName))
	}
	return nil
}

// helmSetup is used for the initial setup of Helm in cluster.
func helmSetup(logger log.FieldLogger, kops *kops.Cmd) error {
	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return errors.Wrap(err, "failed to set up the k8s client")
	}

	logger.Info("Initializing Helm in the cluster")
	err = exec.Command("helm", "--debug", "--kubeconfig", kops.GetKubeConfigPath(), "init", "--upgrade").Run()
	if err != nil {
		return errors.Wrap(err, "failed to initialize Helm in the cluster")
	}

	logger.Info("Creating Tiller service account")
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "tiller"},
	}

	_, err = k8sClient.Clientset.CoreV1().ServiceAccounts("kube-system").Create(serviceAccount)
	if err != nil {
		return errors.Wrap(err, "failed to set up Tiller service account for Helm")
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
		return errors.Wrap(err, "failed to create cluster role bindings")
	}

	logger.Info("Successfully created cluster role bind")

	logger.Info("Upgrade Helm")

	cmd := exec.Command("helm", "--debug", "--kubeconfig", kops.GetKubeConfigPath(), "init", "--service-account", "tiller", "--upgrade")

	logger.WithFields(log.Fields{
		"cmd":  cmd.Path,
		"args": cmd.Args,
	}).Info("Invoking command")

	output, err := cmd.Output()
	if err != nil {
		if len(output) > 0 {
			log.Debugf("Helm output was:\n%s\n", string(output))
		}
		return errors.Wrap(err, "failed to invoke Helm command")
	}

	return nil
}
