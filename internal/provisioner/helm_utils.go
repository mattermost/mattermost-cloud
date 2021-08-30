// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/helm"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// helmDeployment deploys Helm charts.
type helmDeployment struct {
	chartDeploymentName string
	chartName           string
	namespace           string
	setArgument         string
	desiredVersion      *model.HelmUtilityVersion

	cluster         *model.Cluster
	kopsProvisioner *KopsProvisioner
	kops            *kops.Cmd
	logger          log.FieldLogger
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

func (d *helmDeployment) Delete() error {
	logger := d.logger.WithField("helm-delete", d.chartDeploymentName)

	logger.Infof("Deleting helm chart %s", d.chartDeploymentName)
	err := deleteHelmChart(*d, d.kops.GetKubeConfigPath(), logger)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("got an error trying to delete the helm chart %s", d.chartDeploymentName))
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

// upgradeHelmChart is used to upgrade Helm deployments.
func upgradeHelmChart(chart helmDeployment, configPath string, logger log.FieldLogger) error {
	if chart.desiredVersion == nil || chart.desiredVersion.Version() == "" {
		currentVersion, err := chart.Version()
		if err != nil {
			return errors.Wrap(err, "failed to determine current chart version and no desired target version specified")
		}
		if currentVersion.Values() == "" {
			return errors.New("path to values file must not be empty")
		}
		chart.desiredVersion = currentVersion
	}

	censoredPath := chart.desiredVersion.ValuesPath
	defer func(chart *helmDeployment, censoredPath string) {
		// so that we don't store the GitLab secret in the database
		chart.desiredVersion.ValuesPath = censoredPath
	}(&chart, censoredPath)

	var err error
	var cleanup func(string)
	chart.desiredVersion.ValuesPath, cleanup, err = fetchFromGitlabIfNecessary(chart.desiredVersion.ValuesPath)
	if err != nil {
		return errors.Wrap(err, "failed to get values file")
	}
	if cleanup != nil {
		defer cleanup(chart.desiredVersion.ValuesPath)
	}

	arguments := []string{
		"--debug",
		"upgrade",
		chart.chartDeploymentName,
		chart.chartName,
		"--kubeconfig", configPath,
		"-f", chart.desiredVersion.Values(),
		"--namespace", chart.namespace,
		"--install",
		"--create-namespace",
		"--wait",
		"--timeout", "20m",
	}
	if chart.setArgument != "" {
		arguments = append(arguments, "--set", chart.setArgument)
	}
	if chart.desiredVersion.Version() != "" {
		arguments = append(arguments, "--version", chart.desiredVersion.Version())
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

// deleteHelmChart is used to delete Helm charts.
func deleteHelmChart(chart helmDeployment, configPath string, logger log.FieldLogger) error {
	arguments := []string{
		"--debug",
		"delete",
		"--kubeconfig", configPath,
		"--namespace", chart.namespace,
		chart.chartDeploymentName,
	}

	helmClient, err := helm.New(logger)
	if err != nil {
		return errors.Wrap(err, "unable to create helm wrapper")
	}
	defer helmClient.Close()

	err = helmClient.RunGenericCommand(arguments...)
	if err != nil {
		return errors.Wrapf(err, "unable to delete helm chart %s", chart.chartDeploymentName)
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

// HelmListOutput is a struct for holding the unmarshaled
// representation of the output from helm list --output json
type HelmListOutput []helmReleaseJSON

func (l HelmListOutput) containsRelease(name string) bool {
	for _, rel := range l {
		if rel.Name == name {
			return true
		}
	}
	return false
}

func (l HelmListOutput) asSlice() []helmReleaseJSON {
	return l
}

func (l HelmListOutput) asListOutput() *HelmListOutput {
	return &l
}

func (d *helmDeployment) List() (*HelmListOutput, error) {
	arguments := []string{
		"list",
		"--kubeconfig", d.kops.GetKubeConfigPath(),
		"--output", "json",
		"--all-namespaces",
	}

	logger := d.logger.WithFields(log.Fields{
		"cmd": "helm3",
	})

	helmClient, err := helm.New(logger)
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

	var helmList HelmListOutput
	err = json.Unmarshal(rawOutput, &helmList)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal JSON output from helm list")
	}

	return helmList.asListOutput(), nil

}

func (d *helmDeployment) Version() (*model.HelmUtilityVersion, error) {
	output, err := d.List()
	if err != nil {
		return nil, errors.Wrap(err, "while getting Helm Deployment version")
	}

	for _, release := range output.asSlice() {
		if release.Name == d.chartDeploymentName {
			return &model.HelmUtilityVersion{Chart: release.Chart, ValuesPath: d.desiredVersion.Values()}, nil
		}
	}

	return nil, errors.Errorf("unable to get version for chart %s", d.chartDeploymentName)
}

type gitlabValuesFileResponse struct {
	Content string `json:"content"`
}

// fetchFromGitlabIfNecessary returns the path of the values file. If
// this is a local path or a non-Gitlab URL, the path is simply
// returned unchanged. If a Gitlab URL is provided, the values file is
// fetched and stored in the OS's temp dir and the filename of the
// file is returned. If a temp file is created, a cleanup routine will
// be returned as the second return value, otherwise that value will
// be nil
func fetchFromGitlabIfNecessary(path string) (string, func(string), error) {
	gitlabKey := model.GetGitlabToken()
	if gitlabKey == "" {
		return path, nil, nil
	}

	valPathURL, err := url.Parse(path)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to parse Helm values file path or URL")
	}

	// silently allow other public non-Gitlab URLs
	if !strings.HasPrefix(valPathURL.Host, "gitlab") {
		return path, nil, nil
	}

	// if Gitlab, fetch the file using the API
	path = fmt.Sprintf("%s&private_token=%s", path, gitlabKey)

	resp, err := http.Get(path)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to request the values file from Gitlab")
	}
	if resp.StatusCode >= 400 {
		return "", nil, errors.Errorf("request to Gitlab failed with status: %s", resp.Status)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to read body from Gitlab response")
	}

	valuesFileBytes := new(gitlabValuesFileResponse)
	err = json.Unmarshal(body, valuesFileBytes)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to unmarshal JSON in Gitlab response")
	}

	temporaryValuesFile, err := ioutil.TempFile(os.TempDir(), "helm-values-file-")
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to create temporary file for Helm values file")
	}

	content, err := base64.StdEncoding.DecodeString(valuesFileBytes.Content)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to decode base64-encoded YAML file")
	}

	err = ioutil.WriteFile(temporaryValuesFile.Name(), []byte(content), 0600)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to write values file to disk for Helm to read")
	}

	return temporaryValuesFile.Name(), func(path string) {
		if strings.HasPrefix(path, os.TempDir()) {
			os.Remove(path)
		}
	}, nil
}
