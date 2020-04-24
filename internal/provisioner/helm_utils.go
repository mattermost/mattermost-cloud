package provisioner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mattermost/mattermost-cloud/internal/tools/helm"
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
	desiredVersion      string

	cluster         *model.Cluster
	kopsProvisioner *KopsProvisioner
	kops            *kops.Cmd
	logger          log.FieldLogger
}

func installHelm(kops *kops.Cmd, repos map[string]string, logger log.FieldLogger) error {
	logger.Info("Installing Helm")

	err := helmSetup(logger, kops)
	if err != nil {
		return errors.Wrap(err, "unable to install helm")
	}

	wait := 120
	logger.Infof("Waiting up to %d seconds for helm to become ready...", wait)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
	defer cancel()
	err = waitForHelmRunning(ctx, kops.GetKubeConfigPath())
	if err != nil {
		return errors.Wrap(err, "helm didn't start as expected, or we couldn't detect it")
	}

	return nil
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

// helmRepoAdd adds new helm repos
func helmRepoAdd(repoName, repoURL string, logger log.FieldLogger) error {
	logger.Infof("Adding helm repo %s", repoName)
	arguments := []string{
		"repo",
		"add",
		repoName,
		repoURL,
	}

	helmClient, err := helm.New(logger)
	if err != nil {
		return errors.Wrap(err, "unable to create helm wrapper")
	}
	defer helmClient.Close()

	err = helmClient.RunGenericCommand(arguments...)
	if err != nil {
		return errors.Wrapf(err, "unable to add repo %s", repoName)
	}

	return helmRepoUpdate(logger)
}

// helmRepoUpdate updates the helm repos to get latest available charts
func helmRepoUpdate(logger log.FieldLogger) error {
	arguments := []string{
		"repo",
		"update",
	}

	helmClient, err := helm.New(logger)
	if err != nil {
		return errors.Wrap(err, "unable to create helm wrapper")
	}
	defer helmClient.Close()

	err = helmClient.RunGenericCommand(arguments...)
	if err != nil {
		return errors.Wrap(err, "unable to update helm repos")
	}

	return nil
}

// installHelmChart is used to install Helm charts.
func installHelmChart(chart helmDeployment, configPath string, logger log.FieldLogger) error {
	arguments := []string{
		"--debug",
		"install",
		"--kubeconfig", configPath,
		"-f", chart.valuesPath,
		chart.chartName,
		"--namespace", chart.namespace,
		"--name", chart.chartDeploymentName,
	}
	if chart.setArgument != "" {
		arguments = append(arguments, "--set", chart.setArgument)
	}
	if chart.desiredVersion != "" {
		arguments = append(arguments, "--version", chart.desiredVersion)
	}

	helmClient, err := helm.New(logger)
	if err != nil {
		return errors.Wrap(err, "unable to create helm wrapper")
	}
	defer helmClient.Close()

	err = helmClient.RunGenericCommand(arguments...)
	if err != nil {
		return errors.Wrapf(err, "unable to install helm chart %s", chart.chartName)
	}

	return nil
}

// upgradeHelmChart is used to upgrade Helm deployments.
func upgradeHelmChart(chart helmDeployment, configPath string, logger log.FieldLogger) error {
	arguments := []string{
		"--debug",
		"upgrade",
		chart.chartDeploymentName,
		chart.chartName,
		"--kubeconfig", configPath,
		"-f", chart.valuesPath,
		"--namespace", chart.namespace,
		"--install",
	}
	if chart.setArgument != "" {
		arguments = append(arguments, "--set", chart.setArgument)
	}
	if chart.desiredVersion != "" {
		arguments = append(arguments, "--version", chart.desiredVersion)
	}

	helmClient, err := helm.New(logger)
	if err != nil {
		return errors.Wrap(err, "unable to create helm wrapper")
	}
	defer helmClient.Close()

	err = helmClient.RunGenericCommand(arguments...)
	if err != nil {
		return errors.Wrapf(err, "unable to upgrade helm chart %s", chart.chartName)
	}

	return nil
}

type helmReleaseJSON struct {
	Name       string `json:"Name"`
	Revision   int    `json:"Revision"`
	Updated    string `json:"Updated"`
	Status     string `json:"Status"`
	Chart      string `json:"Chart"`
	AppVersion string `json:"AppVersion"`
	Namespace  string `json:"Namespace"`
}

// HelmListOutput is a struct for holding the unmarshaled
// representation of the output from helm list --output json
type HelmListOutput struct {
	Releases []helmReleaseJSON `json:"Releases"`
}

func (d *helmDeployment) List() (*HelmListOutput, error) {
	arguments := []string{
		"list",
		"--kubeconfig", d.kops.GetKubeConfigPath(),
		"--output", "json",
	}

	// TODO: Not using helm client here due to requirement for raw output
	cmd := exec.Command("helm", arguments...)

	logger := d.logger.WithFields(log.Fields{
		"cmd": cmd.Path,
	})

	rawOutput, err := cmd.Output()
	if err != nil {
		if len(rawOutput) > 0 {
			logger.Debugf("Helm output was:\n%s\n", string(rawOutput))
		}
		return nil, err
	}

	output := &HelmListOutput{}
	err = json.Unmarshal(rawOutput, output)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal JSON output from helm list")
	}

	return output, nil

}

func (d *helmDeployment) Version() (string, error) {
	output, err := d.List()
	if err != nil {
		return "", err
	}

	for _, release := range output.Releases {
		if release.Name == d.chartDeploymentName {
			return release.Chart, nil
		}
	}

	return "", errors.Errorf("unable to get version for chart %s", d.chartDeploymentName)
}

// helmSetup is used for the initial setup of Helm in cluster.
func helmSetup(logger log.FieldLogger, kops *kops.Cmd) error {
	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return errors.Wrap(err, "failed to set up the k8s client")
	}

	logger.Info("Creating Tiller service account")
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{Name: "tiller"},
	}

	_, err = k8sClient.Clientset.CoreV1().ServiceAccounts("kube-system").Get("tiller", metav1.GetOptions{})
	if err != nil {
		// need to create cluster role bindings for Tiller since they couldn't be found

		_, err = k8sClient.Clientset.CoreV1().ServiceAccounts("kube-system").Create(serviceAccount)
		if err != nil {
			return errors.Wrap(err, "failed to set up Tiller service account for Helm")
		}

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
	}

	err = helmInit(logger, kops)
	if err != nil {
		return err
	}

	return nil
}

// helmInit calls helm init and doesn't do anything fancy
func helmInit(logger log.FieldLogger, kops *kops.Cmd) error {
	logger.Info("Upgrading Helm")
	helmClient, err := helm.New(logger)
	if err != nil {
		return errors.Wrap(err, "unable to create helm wrapper")
	}
	defer helmClient.Close()

	err = helmClient.RunGenericCommand("--debug", "--kubeconfig", kops.GetKubeConfigPath(), "init", "--service-account", "tiller", "--upgrade")
	if err != nil {
		return errors.Wrap(err, "failed to upgrade helm")
	}

	return nil
}
