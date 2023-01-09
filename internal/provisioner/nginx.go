// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	namespaceNginx = "nginx"
)

type nginx struct {
	awsClient      aws.AWS
	kubeconfigPath string
	logger         log.FieldLogger
	cluster        *model.Cluster
	actualVersion  *model.HelmUtilityVersion
	desiredVersion *model.HelmUtilityVersion
}

func newNginxHandle(version *model.HelmUtilityVersion, cluster *model.Cluster, kubeconfigPath string, awsClient aws.AWS, logger log.FieldLogger) (*nginx, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate NGINX handle with nil logger")
	}
	if cluster == nil {
		return nil, errors.New("cannot create a connection to Nginx if the cluster provided is nil")
	}
	if awsClient == nil {
		return nil, errors.New("cannot create a connection to Nginx if the awsClient provided is nil")
	}
	if kubeconfigPath == "" {
		return nil, errors.New("cannot create utility without kubeconfig")
	}

	return &nginx{
		awsClient:      awsClient,
		kubeconfigPath: kubeconfigPath,
		cluster:        cluster,
		logger:         logger.WithField("cluster-utility", model.NginxCanonicalName),
		desiredVersion: version,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.Nginx,
	}, nil

}

func (n *nginx) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	n.actualVersion = actualVersion
	return nil
}

func (n *nginx) CreateOrUpgrade() error {
	h, err := n.NewHelmDeployment()
	if err != nil {
		return errors.Wrap(err, "failed to generate nginx helm deployment")
	}

	err = h.Update()
	if err != nil {
		return err
	}

	if err = n.updateVersion(h); err != nil {
		return err
	}

	if err = n.addLoadBalancerNameTag(); err != nil {
		n.logger.Errorln(err)
	}

	return nil
}

func (n *nginx) addLoadBalancerNameTag() error {

	endpoint, elbType, err := getElasticLoadBalancerInfo(namespaceNginx, n.logger, n.kubeconfigPath)
	if err != nil {
		return errors.Wrap(err, "couldn't get the loadbalancer endpoint (nginx)")
	}

	if err := addLoadBalancerNameTag(n.awsClient.GetLoadBalancerAPI(elbType), endpoint); err != nil {
		return errors.Wrap(err, "failed to add loadbalancer name tag (nginx)")
	}

	return nil
}

func (n *nginx) DesiredVersion() *model.HelmUtilityVersion {
	return n.desiredVersion
}

func (n *nginx) ActualVersion() *model.HelmUtilityVersion {
	if n.actualVersion == nil {
		return nil
	}

	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(n.actualVersion.Version(), "ingress-nginx-"),
		ValuesPath: n.actualVersion.Values(),
	}
}

func (n *nginx) Destroy() error {
	return nil
}

func (n *nginx) ValuesPath() string {
	if n.desiredVersion == nil {
		return ""
	}
	return n.desiredVersion.Values()
}

func (n *nginx) Migrate() error {
	return nil
}

func (n *nginx) NewHelmDeployment() (*helmDeployment, error) {
	certificate, err := n.awsClient.GetCertificateSummaryByTag(aws.DefaultInstallCertificatesTagKey, aws.DefaultInstallCertificatesTagValue, n.logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrive the AWS ACM")
	}

	if certificate.ARN == nil {
		return nil, errors.New("retrieved certificate does not have ARN")
	}

	clusterResources, err := n.awsClient.GetVpcResources(n.cluster.ID, n.logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrive the VPC information")
	}

	return newHelmDeployment(
		"ingress-nginx/ingress-nginx",
		"nginx",
		namespaceNginx,
		n.kubeconfigPath,
		n.desiredVersion,
		fmt.Sprintf("controller.service.annotations.service\\.beta\\.kubernetes\\.io/aws-load-balancer-ssl-cert=%s,controller.config.proxy-real-ip-cidr=%s", *certificate.ARN, clusterResources.VpcCIDR),
		n.logger,
	), nil
}

func (n *nginx) Name() string {
	return model.NginxCanonicalName
}
