// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utility

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/helm"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// gitlabFetchClient is the HTTP client used to download Helm values files
// from the configured GitLab instance. Redirects are not followed and a
// timeout is applied to keep the supervisor goroutine responsive.
var gitlabFetchClient = &http.Client{
	Timeout: 30 * time.Second,
	CheckRedirect: func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

const (
	defaultKubeConfigPath            = ""
	defaultHelmDeploymentSetArgument = ""

	// maxGitlabResponseSize caps the response body read from the GitLab
	// values endpoint to keep memory usage bounded.
	maxGitlabResponseSize = 10 << 20 // 10 MiB
)

// helmDeployment deploys Helm charts.
type helmDeployment struct {
	chartDeploymentName string
	chartName           string
	namespace           string
	setArgument         string
	desiredVersion      *model.HelmUtilityVersion

	kubeconfigPath string
	logger         log.FieldLogger
}

func newHelmDeployment(
	chartName, chartDeploymentName, namespace, kubeConfigPath string,
	desiredVersion *model.HelmUtilityVersion,
	setArgument string,
	logger log.FieldLogger,
) *helmDeployment {
	return &helmDeployment{
		chartName:           chartName,
		chartDeploymentName: chartDeploymentName,
		namespace:           namespace,
		kubeconfigPath:      kubeConfigPath,
		desiredVersion:      desiredVersion,
		setArgument:         setArgument,
		logger:              logger,
	}
}

func (d *helmDeployment) Update() error {
	logger := d.logger.WithField("helm-update", d.chartName)

	logger.Infof("Refreshing helm chart %s -- may trigger service upgrade", d.chartName)
	err := upgradeHelmChart(*d, d.kubeconfigPath, logger)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("got an error trying to upgrade the helm chart %s", d.chartName))
	}
	return nil
}

func (d *helmDeployment) Delete() error {
	logger := d.logger.WithField("helm-delete", d.chartDeploymentName)

	// Ensure the chart is present before deletion
	exists, err := d.Exists()
	if err != nil {
		return err
	}

	if !exists {
		logger.Warnf("chart %s not present, assuming already deleted", d.chartDeploymentName)
		return nil
	}

	err = deleteHelmChart(*d, d.kubeconfigPath, logger)
	if err != nil {
		return errors.Wrapf(err, "got an error trying to delete the helm chart %s", d.chartDeploymentName)
	}

	return nil
}

func (d *helmDeployment) Exists() (bool, error) {
	list, err := d.List()
	if err != nil {
		return false, errors.Wrap(err, "failed to list helm charts")
	}

	for _, chart := range list.asSlice() {
		if chart.Name == d.chartDeploymentName && chart.Namespace == d.namespace {
			return true, nil
		}
	}

	return false, nil
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

	helmClient, err := helm.New(configPath, logger)
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
		"uninstall",
		chart.chartDeploymentName,
		"--namespace", chart.namespace,
		"--wait",
		"--debug",
	}

	helmClient, err := helm.New(configPath, logger)
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

func (l HelmListOutput) asSlice() []helmReleaseJSON {
	return l
}

func (l HelmListOutput) asListOutput() *HelmListOutput {
	return &l
}

func (d *helmDeployment) List() (*HelmListOutput, error) {
	arguments := []string{
		"list",
		"--output", "json",
		"--all-namespaces",
	}

	logger := d.logger.WithFields(log.Fields{
		"cmd": "helm3",
	})

	helmClient, err := helm.New(d.kubeconfigPath, logger)
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

// isConfiguredGitlabHost reports whether valPathURL points at the GitLab
// instance configured via --utilities-git-url. The hostname comparison is
// exact and case-insensitive.
func isConfiguredGitlabHost(valPathURL *url.URL) bool {
	configured := model.GetGitopsRepoURL()
	if configured == "" {
		return false
	}
	configuredURL, err := url.Parse(configured)
	if err != nil || configuredURL.Hostname() == "" {
		return false
	}
	return strings.EqualFold(valPathURL.Hostname(), configuredURL.Hostname())
}

// fetchFromGitlabIfNecessary returns the path of the values file. If
// this is a local path or a non-Gitlab URL, the path is simply
// returned unchanged. If the configured Gitlab host is provided over
// HTTPS, the values file is fetched and stored in the OS's temp dir and
// the filename of the file is returned. If a temp file is created, a
// cleanup routine will be returned as the second return value,
// otherwise that value will be nil
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
	if !isConfiguredGitlabHost(valPathURL) {
		return path, nil, nil
	}

	if !strings.EqualFold(valPathURL.Scheme, "https") {
		return "", nil, errors.New("Gitlab values file URL must use HTTPS")
	}

	req, err := http.NewRequest(http.MethodGet, path, nil)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to build Gitlab request")
	}
	req.Header.Set("PRIVATE-TOKEN", gitlabKey)

	resp, err := gitlabFetchClient.Do(req)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to request the values file from Gitlab")
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", nil, errors.Errorf("request to Gitlab failed with status: %s", resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxGitlabResponseSize))
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to read body from Gitlab response")
	}

	valuesFileBytes := new(gitlabValuesFileResponse)
	err = json.Unmarshal(body, valuesFileBytes)
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to unmarshal JSON in Gitlab response")
	}

	temporaryValuesFile, err := os.CreateTemp(os.TempDir(), "helm-values-file-")
	if err != nil {
		return "", nil, errors.Wrap(err, "failed to create temporary file for Helm values file")
	}
	tmpPath := temporaryValuesFile.Name()
	temporaryValuesFile.Close()

	content, err := base64.StdEncoding.DecodeString(valuesFileBytes.Content)
	if err != nil {
		os.Remove(tmpPath)
		return "", nil, errors.Wrap(err, "failed to decode base64-encoded YAML file")
	}

	if err := os.WriteFile(tmpPath, content, 0600); err != nil {
		os.Remove(tmpPath)
		return "", nil, errors.Wrap(err, "failed to write values file to disk for Helm to read")
	}

	return tmpPath, func(path string) {
		if strings.HasPrefix(path, os.TempDir()) {
			os.Remove(path)
		}
	}, nil
}
