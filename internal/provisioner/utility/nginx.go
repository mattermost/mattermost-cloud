// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utility

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
	provisioner    string
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
		provisioner:    cluster.Provisioner,
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
	h, err := n.newHelmDeployment()
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

	if err := addLoadBalancerNameTag(n.awsClient.GetLoadBalancerAPIByType(elbType), endpoint); err != nil {
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
	helm, err := n.newHelmDeployment()
	if err != nil {
		return err
	}
	return helm.Delete()
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

func (n *nginx) newHelmDeployment() (*helmDeployment, error) {
	certificate, err := n.awsClient.GetCertificateSummaryByTag(aws.DefaultInstallCertificatesTagKey, aws.DefaultInstallCertificatesTagValue, n.logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to retrive the AWS ACM")
	}

	if certificate.ARN == nil {
		return nil, errors.New("retrieved certificate does not have ARN")
	}

	setArguments := []string{
		fmt.Sprintf("controller.service.annotations.service\\.beta\\.kubernetes\\.io/aws-load-balancer-ssl-cert=%s", *certificate.ARN),
	}

	vpc, err := n.awsClient.GetClaimedVPC(n.cluster.ID, n.logger)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to perform VPC lookup for cluster %s", n.cluster.ID)
	}

	if vpc != "" {
		setArguments = append(setArguments, fmt.Sprintf("controller.service.annotations.service\\.beta\\.kubernetes\\.io/aws-load-balancer-additional-resource-tags=VpcID=%s", vpc))
	}

	if n.provisioner == model.ProvisionerEKS {
		// Calico networking cannot currently be installed on the EKS control plane nodes.
		// As a result the control plane nodes will not be able to initiate network connections to Calico pods.
		// As a workaround, trusted pods that require control plane nodes to connect to them,
		// such as those implementing admission controller webhooks, can include hostNetwork:true in their pod spec.
		// See https://docs.tigera.io/calico/3.25/getting-started/kubernetes/managed-public-cloud/eks

		// setArguments = append(setArguments, "controller.hostNetwork=true")

		// hostNetwork can cause port conflict, that's why we need to use DaemonSet
		// setArguments = append(setArguments, "controller.kind=DaemonSet")

		setArguments = append(setArguments, "controller.admissionWebhooks.enabled=false")
	}

	return newHelmDeployment(
		"ingress-nginx/ingress-nginx",
		"nginx",
		namespaceNginx,
		n.kubeconfigPath,
		n.desiredVersion,
		strings.Join(setArguments, ","),
		n.logger,
	), nil
}

func (n *nginx) Name() string {
	return model.NginxCanonicalName
}
