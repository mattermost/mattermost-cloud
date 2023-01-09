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
	namespaceNginxInternal           = "nginx-internal"
	chartDeploymentNameNginxInternal = "nginx-internal"
	chartNameNginxInternal           = "ingress-nginx/ingress-nginx"
)

type nginxInternal struct {
	awsClient      aws.AWS
	kubeconfigPath string
	logger         log.FieldLogger
	cluster        *model.Cluster
	actualVersion  *model.HelmUtilityVersion
	desiredVersion *model.HelmUtilityVersion
}

func newNginxInternalHandle(version *model.HelmUtilityVersion, cluster *model.Cluster, kubeconfigPath string, awsClient aws.AWS, logger log.FieldLogger) (*nginxInternal, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate NGINX INTERNAL handle with nil logger")
	}

	if cluster == nil {
		return nil, errors.New("cannot create a connection to Nginx internal if the cluster provided is nil")
	}
	if awsClient == nil {
		return nil, errors.New("cannot create a connection to Nginx internal if the awsClient provided is nil")
	}
	if kubeconfigPath == "" {
		return nil, errors.New("cannot create utility without kubeconfig")
	}

	return &nginxInternal{
		awsClient:      awsClient,
		kubeconfigPath: kubeconfigPath,
		cluster:        cluster,
		logger:         logger.WithField("cluster-utility", model.NginxInternalCanonicalName),
		desiredVersion: version,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.NginxInternal,
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
	h, err := n.NewHelmDeployment(true)
	if err != nil {
		return errors.Wrap(err, "failed to generate nginx internal helm deployment")
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

func (n *nginxInternal) addLoadBalancerNameTag() error {

	endpoint, elbType, err := getElasticLoadBalancerInfo(namespaceNginxInternal, n.logger, n.kubeconfigPath)
	if err != nil {
		return errors.Wrap(err, "couldn't get the loadbalancer endpoint (nginx-internal)")
	}

	if err := addLoadBalancerNameTag(n.awsClient.GetLoadBalancerAPI(elbType), endpoint); err != nil {
		return errors.Wrap(err, "failed to add loadbalancer name tag (nginx-internal)")
	}

	return nil
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
	helm, err := n.NewHelmDeployment(false)
	if err != nil {
		return err
	}
	return helm.Delete()
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

func (n *nginxInternal) NewHelmDeployment(withArguments bool) (*helmDeployment, error) {
	var setArguments string
	if withArguments {
		certificate, err := n.awsClient.GetCertificateSummaryByTag(aws.DefaultInstallPrivateCertificatesTagKey, aws.DefaultInstallPrivateCertificatesTagValue, n.logger)
		if err != nil {
			return nil, errors.Wrap(err, "failed to retrive the AWS Private ACM")
		}

		if certificate.ARN == nil {
			return nil, errors.New("retrieved certificate does not have ARN")
		}

		setArguments = fmt.Sprintf("controller.service.annotations.service\\.beta\\.kubernetes\\.io/aws-load-balancer-ssl-cert=%s", *certificate.ARN)
	}

	return newHelmDeployment(
		chartNameNginxInternal,
		chartDeploymentNameNginxInternal,
		namespaceNginxInternal,
		n.kubeconfigPath,
		n.desiredVersion,
		setArguments,
		n.logger,
	), nil
}

func (n *nginxInternal) Name() string {
	return model.NginxInternalCanonicalName
}

func addLoadBalancerNameTag(elbClient aws.ELB, hostname string) error {
	if hostname == "" {
		return errors.New("cannot add loadbalancer name tag if hostname is empty")
	}

	parts := strings.Split(hostname, "-")
	loadbalancerName := parts[0]

	resource, err := elbClient.GetLoadBalancerResource(loadbalancerName)
	if err != nil {
		return errors.Wrap(err, "failed to get loadbalancer ARN")
	}

	err = elbClient.TagLoadBalancer(resource, map[string]string{
		"Name": loadbalancerName,
	})
	if err != nil {
		return errors.Wrap(err, "failed to tag loadbalancer")
	}

	return nil
}
