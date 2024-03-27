// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utility

import (
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/argocd"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/git"
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

func newFluentbitOrUnmanagedHandle(cluster *model.Cluster, kubeconfigPath string, tempDir string, awsClient aws.AWS, gitClient git.Client, argocdClient argocd.Client, logger log.FieldLogger) (Utility, error) {
	desired := cluster.DesiredUtilityVersion(model.FluentbitCanonicalName)
	actual := cluster.ActualUtilityVersion(model.FluentbitCanonicalName)

	if model.UtilityIsUnmanaged(desired, actual) {
		return newUnmanagedHandle(model.FluentbitCanonicalName, kubeconfigPath, tempDir, []string{}, cluster, awsClient, gitClient, argocdClient, logger), nil
	}

	fluentbit := newFluentbitHandle(cluster, desired, kubeconfigPath, awsClient, logger)
	err := fluentbit.validate()
	if err != nil {
		return nil, errors.Wrap(err, "fluentbit utility config is invalid")
	}

	return fluentbit, nil
}

func newFluentbitHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, kubeconfigPath string, awsClient aws.AWS, logger log.FieldLogger) *fluentbit {
	return &fluentbit{
		awsClient:      awsClient,
		kubeconfigPath: kubeconfigPath,
		logger:         logger.WithField("cluster-utility", model.FluentbitCanonicalName),
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.Fluentbit,
	}
}

func (f *fluentbit) validate() error {
	if f.kubeconfigPath == "" {
		return errors.New("kubeconfig path cannot be empty")
	}
	if f.awsClient == nil {
		return errors.New("awsClient cannot be nil")
	}

	return nil
}

func (f *fluentbit) Destroy() error {
	helm := f.newHelmDeployment()
	return helm.Delete()
}

func (f *fluentbit) Migrate() error {
	return nil
}

func (f *fluentbit) CreateOrUpgrade() error {
	h := f.newHelmDeployment()

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

func (f *fluentbit) newHelmDeployment() *helmDeployment {
	return newHelmDeployment(
		"fluent/fluent-bit",
		"fluent-bit",
		"fluent-bit",
		f.kubeconfigPath,
		f.desiredVersion,
		defaultHelmDeploymentSetArgument,
		f.logger,
	)
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
