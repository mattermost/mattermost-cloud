// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/helm"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// HelmUtilsManager is a wrapper for Helm related operations based on used version
type HelmUtilsManager struct {
	deprecatedHelm2    bool
	helmClientProvider func(logger log.FieldLogger) (*helm.Cmd, error)
}

// NewHelmUtilsManager creates new HelmUtilsManager
func NewHelmUtilsManager(useHelm2 bool) *HelmUtilsManager {
	provider := helm.NewV3
	if useHelm2 {
		provider = helm.New
	}

	return &HelmUtilsManager{
		deprecatedHelm2:    useHelm2,
		helmClientProvider: provider,
	}
}

// addUpgradeArgs adds version specific arguments for `helm upgrade` command.
func (um *HelmUtilsManager) addUpgradeArgs(args []string) []string {
	if !um.deprecatedHelm2 {
		args = append(args, "--create-namespace")
	}
	return args
}

// addListArgs adds version specific arguments for `helm list` command.
func (um *HelmUtilsManager) addListArgs(args []string) []string {
	if !um.deprecatedHelm2 {
		args = append(args, "--all-namespaces")
	}
	return args
}

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

func (um *HelmUtilsManager) installHelm(kops *kops.Cmd, logger log.FieldLogger) error {
	logger.Info("Installing Helm")

	if !um.deprecatedHelm2 {
		return nil
	}

	err := um.helmSetup(logger, kops)
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

func (d *helmDeployment) Update(helmUtilManager *HelmUtilsManager) error {
	logger := d.logger.WithField("helm-update", d.chartName)

	logger.Infof("Refreshing helm chart %s -- may trigger service upgrade", d.chartName)
	err := helmUtilManager.upgradeHelmChart(*d, d.kops.GetKubeConfigPath(), logger)
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
func (um *HelmUtilsManager) helmRepoAdd(repoName, repoURL string, logger log.FieldLogger) error {
	logger.Infof("Adding helm repo %s", repoName)
	arguments := []string{
		"repo",
		"add",
		repoName,
		repoURL,
	}

	helmClient, err := um.helmClientProvider(logger)
	if err != nil {
		return errors.Wrap(err, "unable to create helm wrapper")
	}
	defer helmClient.Close()

	err = helmClient.RunGenericCommand(arguments...)
	if err != nil {
		return errors.Wrapf(err, "unable to add repo %s", repoName)
	}

	return um.helmRepoUpdate(logger)
}

// helmRepoUpdate updates the helm repos to get latest available charts
func (um *HelmUtilsManager) helmRepoUpdate(logger log.FieldLogger) error {
	arguments := []string{
		"repo",
		"update",
	}

	helmClient, err := um.helmClientProvider(logger)
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

// upgradeHelmChart is used to upgrade Helm deployments.
func (um *HelmUtilsManager) upgradeHelmChart(chart helmDeployment, configPath string, logger log.FieldLogger) error {
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
	arguments = um.addUpgradeArgs(arguments)
	if chart.setArgument != "" {
		arguments = append(arguments, "--set", chart.setArgument)
	}
	if chart.desiredVersion != "" {
		arguments = append(arguments, "--version", chart.desiredVersion)
	}

	helmClient, err := um.helmClientProvider(logger)
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
	Name       string `json:"name"`
	Revision   string `json:"revision"`
	Updated    string `json:"updated"`
	Status     string `json:"status"`
	Chart      string `json:"chart"`
	AppVersion string `json:"appVersion"`
	Namespace  string `json:"namespace"`
}

type helmReleaseList interface {
	asListOutput() *HelmListOutput
}

// HelmListOutput is a struct for holding the unmarshaled
// representation of the output from helm list --output json
type HelmListOutput []helmReleaseJSON

func (l HelmListOutput) asSlice() []helmReleaseJSON {
	return l
}

func (l HelmListOutput) asListOutput() *HelmListOutput {
	return &l
}

func (d *helmDeployment) List(helmUtilManager *HelmUtilsManager) (*HelmListOutput, error) {
	arguments := []string{
		"list",
		"--kubeconfig", d.kops.GetKubeConfigPath(),
		"--output", "json",
	}
	arguments = helmUtilManager.addListArgs(arguments)

	logger := d.logger.WithFields(log.Fields{
		"cmd": "helm",
	})

	helmClient, err := helmUtilManager.helmClientProvider(logger)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create helm wrapper")
	}
	defer helmClient.Close()

	rawOutput, err := helmClient.RunCommandRaw(arguments...)
	if err != nil {
		if len(rawOutput) > 0 {
			logger.Debugf("Helm output was:\n%s\n", string(rawOutput))
		}
		return nil, errors.Wrap(err, "while listing Helm Releases")
	}

	var helmList helmReleaseList
	if helmUtilManager.deprecatedHelm2 {
		helmList = &Helm2ListOutput{}
	} else {
		helmList = &HelmListOutput{}
	}

	err = json.Unmarshal(rawOutput, helmList)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal JSON output from helm list")
	}

	return helmList.asListOutput(), nil

}

func (d *helmDeployment) Version(helmUtilManager *HelmUtilsManager) (string, error) {
	output, err := d.List(helmUtilManager)
	if err != nil {
		return "", errors.Wrap(err, "while getting Helm Deployment version")
	}

	for _, release := range output.asSlice() {
		if release.Name == d.chartDeploymentName {
			return release.Chart, nil
		}
	}

	return "", errors.Errorf("unable to get version for chart %s", d.chartDeploymentName)
}
