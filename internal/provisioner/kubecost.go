// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"github.com/mattermost/mattermost-cloud/k8s"
	"os"
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type kubecost struct {
	awsClient      aws.AWS
	environment    string
	provisioner    *KopsProvisioner
	kops           *kops.Cmd
	cluster        *model.Cluster
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func newKubecostHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, provisioner *KopsProvisioner, awsClient aws.AWS, kops *kops.Cmd, logger log.FieldLogger) (*kubecost, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate Kubecost handle with nil logger")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to Kubecost if the provisioner provided is nil")
	}

	if kops == nil {
		return nil, errors.New("cannot create a connection to Kubecost if the Kops command provided is nil")
	}

	return &kubecost{
		awsClient:      awsClient,
		environment:    awsClient.GetCloudEnvironmentName(),
		provisioner:    provisioner,
		kops:           kops,
		cluster:        cluster,
		logger:         logger.WithField("cluster-utility", model.KubecostCanonicalName),
		desiredVersion: desiredVersion,
	}, nil

}

func (k *kubecost) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	k.actualVersion = actualVersion
	return nil
}

func (k *kubecost) ValuesPath() string {
	if k.desiredVersion == nil {
		return ""
	}
	return k.desiredVersion.Values()
}

func (k *kubecost) CreateOrUpgrade() error {
	logger := k.logger.WithField("kubecost-action", "create")

	k8sClient, err := k8s.NewFromFile(k.kops.GetKubeConfigPath(), logger)
	if err != nil {
		return errors.Wrap(err, "failed to set up the k8s client")
	}
	_, err = k8sClient.CreateOrUpdateNamespace("kubecost")
	if err != nil {
		return errors.Wrapf(err, "failed to create the kubecost namespace")
	}

	h := k.NewHelmDeployment()

	err = h.Update()
	if err != nil {
		return err
	}

	err = k.updateVersion(h)
	return err
}

func (k *kubecost) DesiredVersion() *model.HelmUtilityVersion {
	return k.desiredVersion
}

func (k *kubecost) ActualVersion() *model.HelmUtilityVersion {
	if k.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(k.actualVersion.Version(), "cost-analyzer-"),
		ValuesPath: k.actualVersion.Values(),
	}
}

func (k *kubecost) Destroy() error {
	return nil
}

func (k *kubecost) Migrate() error {
	return nil
}

func (k *kubecost) NewHelmDeployment() *helmDeployment {
	kubecostToken :=""
	if len(os.Getenv(model.KubecostToken)) > 0 {
		kubecostToken = "kubecostToken="+os.Getenv(model.KubecostToken)
	}

	return &helmDeployment{
		chartDeploymentName: "cost-analyzer",
		chartName:           "kubecost/cost-analyzer",
		namespace:           "kubecost",
		kopsProvisioner:     k.provisioner,
		kops:                k.kops,
		setArgument:         kubecostToken,
		logger:              k.logger,
		desiredVersion:      k.desiredVersion,
	}
}

func (k *kubecost) Name() string {
	return model.KubecostCanonicalName
}
