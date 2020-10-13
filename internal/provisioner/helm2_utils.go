// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

// TODO: this file can be removed after full migration to Helm 3

import (
	"encoding/json"
	"github.com/mattermost/mattermost-cloud/internal/tools/helm"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strconv"
)

type helm2ReleaseJSON struct {
	Name       string `json:"Name"`
	Revision   int    `json:"Revision"`
	Updated    string `json:"Updated"`
	Status     string `json:"Status"`
	Chart      string `json:"Chart"`
	AppVersion string `json:"AppVersion"`
	Namespace  string `json:"Namespace"`
}

func (r helm2ReleaseJSON) toHelmRelease() helmReleaseJSON {
	return helmReleaseJSON{
		Name:       r.Name,
		Revision:   strconv.Itoa(r.Revision),
		Updated:    r.Updated,
		Status:     r.Status,
		Chart:      r.Chart,
		AppVersion: r.AppVersion,
		Namespace:  r.Namespace,
	}
}

// Helm2ListOutput is a struct for holding the unmarshaled
// representation of the output from helm list --output json (for Helm 2)
type Helm2ListOutput struct {
	Releases []helm2ReleaseJSON `json:"Releases"`
}

func (l Helm2ListOutput) asListOutput() *HelmListOutput {
	out := make([]helmReleaseJSON, 0, len(l.Releases))

	for _, rel := range l.Releases {
		out = append(out, rel.toHelmRelease())
	}
	list := HelmListOutput(out)
	return &list
}

func (d *helmDeployment) ListV2() (*HelmListOutput, error) {
	arguments := []string{
		"list",
		"--kubeconfig", d.kops.GetKubeConfigPath(),
		"--output", "json",
	}

	logger := d.logger.WithFields(log.Fields{
		"cmd": "helm",
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

	if len(rawOutput) == 0 {
		return &HelmListOutput{}, nil
	}

	var helmList Helm2ListOutput
	err = json.Unmarshal(rawOutput, &helmList)
	if err != nil {
		return nil, errors.Wrap(err, "unable to unmarshal JSON output from helm list")
	}

	return helmList.asListOutput(), nil
}
