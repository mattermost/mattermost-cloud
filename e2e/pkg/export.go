// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//+build e2e

package pkg

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/pkg/errors"
)

// SQLSettings is struct copied from Mattermost Server
type SQLSettings struct {
	DriverName                  *string
	DataSource                  *string
	DataSourceReplicas          []string
	DataSourceSearchReplicas    []string
	MaxIdleConns                *int
	ConnMaxLifetimeMilliseconds *int
	MaxOpenConns                *int
	Trace                       *bool
	AtRestEncryptKey            *string
	QueryTimeout                *int
}

// GetConnectionString fetches the connection string from cluster request's config.
func GetConnectionString(client *model.Client, clusterInstallationID string) (string, error) {
	out, err := client.RunMattermostCLICommandOnClusterInstallation(clusterInstallationID, []string{"config", "show", "--json"})
	if err != nil {
		return "", errors.Wrap(err, "while execing config show")
	}

	settings := struct {
		SQLSettings SQLSettings
	}{}

	err = json.Unmarshal(out, &settings)
	if err != nil {
		return "", errors.Wrap(err, "while unmarshalling sql setting")
	}

	return *settings.SQLSettings.DataSource, nil
}

// BulkLine represents bulk export line.
type BulkLine struct {
	Type string `json:"type"`
}

// ExportStats are statistics of bulk export.
type ExportStats struct {
	Teams          int
	Channels       int
	Users          int
	Posts          int
	DirectChannels int
	DirectPosts    int
}

// GetBulkExportStats runs mattermost bulk export on the cluster request and counts teams, channels, users and posts.
func GetBulkExportStats(client *model.Client, kubeClient kubernetes.Interface, clusterInstallationID, installationID string, logger logrus.FieldLogger) (ExportStats, error) {
	fileName := fmt.Sprintf("export-ci-%s.json", clusterInstallationID)

	_, err := client.RunMattermostCLICommandOnClusterInstallation(clusterInstallationID, []string{"export", "bulk", fileName})
	if err != nil {
		return ExportStats{}, errors.Wrap(err, "while execing export csv")
	}

	podClient := kubeClient.CoreV1().Pods(installationID)

	pods, err := podClient.List(context.Background(), metav1.ListOptions{
		LabelSelector: "app=mattermost",
	})
	if err != nil {
		return ExportStats{}, errors.Wrap(err, "while getting pods")
	}

	destination := fileName
	defer func() {
		err := os.Remove(destination)
		if err != nil {
			logger.WithError(err).Warnf("failed to cleanup file %s", destination)
		}
	}()

	// The solution with `kubectl cp` is pretty hacky, but should be fine for the purpose of tests.

	// File will exist on only one pod.
	// If file does not exist kubectl cp exits with 0 code but does not change local file.
	for _, pod := range pods.Items {
		copyFrom := fmt.Sprintf("%s/%s:/mattermost/%s", pod.Namespace, pod.Name, fileName)
		cmd := exec.Command("kubectl", "cp", copyFrom, destination)
		err := cmd.Run()
		if err != nil {
			return ExportStats{}, errors.Wrap(err, "while copying import file from pod")
		}
	}

	file, err := os.Open(destination)
	if err != nil {
		return ExportStats{}, errors.Wrap(err, "failed to open export file")
	}
	defer file.Close()

	exportStats := ExportStats{}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := BulkLine{}
		err := json.Unmarshal(scanner.Bytes(), &line)
		if err != nil {
			return ExportStats{}, errors.Wrap(err, "while unmarshalling export line")
		}

		switch line.Type {
		case "team":
			exportStats.Teams++
		case "channel":
			exportStats.Channels++
		case "post":
			exportStats.Posts++
		case "user":
			exportStats.Users++
		case "direct_channel":
			exportStats.DirectChannels++
		case "direct_post":
			exportStats.DirectPosts++
		}
	}
	if err := scanner.Err(); err != nil {
		return ExportStats{}, errors.Wrap(err, "error scaning export file")
	}

	return exportStats, nil
}
