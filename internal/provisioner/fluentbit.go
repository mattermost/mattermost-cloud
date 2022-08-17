// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type fluentbit struct {
	awsClient      aws.AWS
	kubeconfigPath string
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func newFluentbitHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, kubeconfigPath string, awsClient aws.AWS, logger log.FieldLogger) (*fluentbit, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate Fluentbit handle with nil logger")
	}

	if awsClient == nil {
		return nil, errors.New("cannot create a connection to Fluentbit if the awsClient provided is nil")
	}

	return &fluentbit{
		awsClient:      awsClient,
		kubeconfigPath: kubeconfigPath,
		logger:         logger.WithField("cluster-utility", model.FluentbitCanonicalName),
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.Fluentbit,
	}, nil
}

func (f *fluentbit) Destroy() error {
	return nil
}

func (f *fluentbit) Migrate() error {
	return nil
}

func (f *fluentbit) CreateOrUpgrade() error {
	h := f.NewHelmDeployment()

	err := h.Update()
	if err != nil {
		return err
	}

	err = f.updateVersion(h)
	return err
}

func (f *fluentbit) DesiredVersion() *model.HelmUtilityVersion {
	return f.desiredVersion
}

func (f *fluentbit) ActualVersion() *model.HelmUtilityVersion {
	if f.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(f.actualVersion.Version(), "fluent-bit-"),
		ValuesPath: f.actualVersion.Values(),
	}
}

func (f *fluentbit) Name() string {
	return model.FluentbitCanonicalName
}

func (f *fluentbit) NewHelmDeployment() *helmDeployment {
	return &helmDeployment{
		kubeconfigPath:      f.kubeconfigPath,
		chartDeploymentName: "fluent-bit",
		chartName:           "fluent/fluent-bit",
		namespace:           "fluent-bit",
		logger:              f.logger,
		desiredVersion:      f.desiredVersion,
	}
}

func (f *fluentbit) ValuesPath() string {
	if f.desiredVersion == nil {
		return ""
	}
	return f.desiredVersion.Values()
}

func (f *fluentbit) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	f.actualVersion = actualVersion
	return nil
}
