// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type nginxInternal struct {
	awsClient      aws.AWS
	provisioner    *KopsProvisioner
	kops           *kops.Cmd
	logger         log.FieldLogger
	cluster        *model.Cluster
	actualVersion  *model.HelmUtilityVersion
	desiredVersion *model.HelmUtilityVersion
}

func newNginxInternalHandle(version *model.HelmUtilityVersion, cluster *model.Cluster, provisioner *KopsProvisioner, awsClient aws.AWS, kops *kops.Cmd, logger log.FieldLogger) (*nginxInternal, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate NGINX INTERNAL handle with nil logger")
	}

	if cluster == nil {
		return nil, errors.New("cannot create a connection to Nginx internal if the cluster provided is nil")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to Nginx internal if the provisioner provided is nil")
	}

	if awsClient == nil {
		return nil, errors.New("cannot create a connection to Nginx internal if the awsClient provided is nil")
	}

	if kops == nil {
		return nil, errors.New("cannot create a connection to Nginx internal if the Kops command provided is nil")
	}

	return &nginxInternal{
		awsClient:      awsClient,
		provisioner:    provisioner,
		kops:           kops,
		cluster:        cluster,
		logger:         logger.WithField("cluster-utility", model.NginxCanonicalName),
		desiredVersion: version,
	}, nil
}

func (n *nginxInternal) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	n.actualVersion = actualVersion
	return nil
}

func (n *nginxInternal) CreateOrUpgrade() error {
	h, err := n.NewHelmDeployment()
	if err != nil {
		return errors.Wrap(err, "failed to generate nginx internal helm deployment")
	}

	err = h.Update()
	if err != nil {
		return err
	}

	err = n.updateVersion(h)
	return err
}

func (n *nginxInternal) DesiredVersion() *model.HelmUtilityVersion {
	return n.desiredVersion
}

func (n *nginxInternal) ActualVersion() *model.HelmUtilityVersion {
	if n.actualVersion == nil {
		return nil
	}

	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(n.actualVersion.Version(), "ingress-nginx-"),
		ValuesPath: n.actualVersion.Values(),
	}
}

func (n *nginxInternal) Destroy() error {
	return nil
}

func (n *nginxInternal) ValuesPath() string {
	if n.desiredVersion == nil {
		return ""
	}
	return n.desiredVersion.Values()
}

func (n *nginxInternal) Migrate() error {
	return nil
}

func (n *nginxInternal) NewHelmDeployment() (*helmDeployment, error) {

	awsACMPrivateCert, err := n.awsClient.GetCertificateSummaryByTag(aws.DefaultInstallPrivateCertificatesTagKey, aws.DefaultInstallPrivateCertificatesTagValue, n.logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrive the AWS Private ACM")
	}

	return &helmDeployment{
		chartDeploymentName: "nginx-internal",
		chartName:           "ingress-nginx/ingress-nginx",
		namespace:           "nginx-internal",
		setArgument:         fmt.Sprintf("controller.service.annotations.service\\.beta\\.kubernetes\\.io/aws-load-balancer-ssl-cert=%s", *awsACMPrivateCert.CertificateArn),
		desiredVersion:      n.desiredVersion,

		cluster:         n.cluster,
		kopsProvisioner: n.provisioner,
		kops:            n.kops,
		logger:          n.logger,
	}, nil
}

func (n *nginxInternal) Name() string {
	return model.NginxInternalCanonicalName
}
