// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utility

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/provisioner/prometheus"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type prometheusOperator struct {
	awsClient          aws.AWS
	cluster            *model.Cluster
	allowCIDRRangeList []string
	kubeconfigPath     string
	logger             log.FieldLogger
	desiredVersion     *model.HelmUtilityVersion
	actualVersion      *model.HelmUtilityVersion
}

func newPrometheusOperatorOrUnmanagedHandle(cluster *model.Cluster, kubeconfigPath string, allowCIDRRangeList []string, awsClient aws.AWS, logger log.FieldLogger) (Utility, error) {
	desired := cluster.DesiredUtilityVersion(model.PrometheusOperatorCanonicalName)
	actual := cluster.ActualUtilityVersion(model.PrometheusOperatorCanonicalName)

	if model.UtilityIsUnmanaged(desired, actual) {
		return newUnmanagedHandle(model.PrometheusOperatorCanonicalName, kubeconfigPath, allowCIDRRangeList, cluster, awsClient, logger), nil
	}

	prometheusOperator := newPrometheusOperatorHandle(cluster, desired, kubeconfigPath, allowCIDRRangeList, awsClient, logger)
	err := prometheusOperator.validate()
	if err != nil {
		return nil, errors.Wrap(err, "prometheus operator utility config is invalid")
	}

	return prometheusOperator, nil
}

func newPrometheusOperatorHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, kubeconfigPath string, allowCIDRRangeList []string, awsClient aws.AWS, logger log.FieldLogger) *prometheusOperator {
	return &prometheusOperator{
		awsClient:          awsClient,
		cluster:            cluster,
		allowCIDRRangeList: allowCIDRRangeList,
		kubeconfigPath:     kubeconfigPath,
		logger:             logger.WithField("cluster-utility", model.PrometheusOperatorCanonicalName),
		desiredVersion:     desiredVersion,
		actualVersion:      cluster.UtilityMetadata.ActualVersions.PrometheusOperator,
	}
}

func (p *prometheusOperator) validate() error {
	if p.kubeconfigPath == "" {
		return errors.New("kubeconfig path cannot be empty")
	}
	if p.awsClient == nil {
		return errors.New("awsClient cannot be nil")
	}

	return nil
}

func (p *prometheusOperator) CreateOrUpgrade() error {
	logger := p.logger.WithField("prometheus-action", "create")

	secretData := map[string]interface{}{
		"type": "s3",
		"config": map[string]interface{}{
			"bucket":       fmt.Sprintf("cloud-%s-prometheus-metrics", p.awsClient.GetCloudEnvironmentName()),
			"endpoint":     p.awsClient.GetS3RegionURL(),
			"aws_sdk_auth": true,
			"sse_config": map[string]string{
				"type": "SSE-S3",
			},
		},
	}

	secret, err := yaml.Marshal(secretData)
	if err != nil {
		return errors.Wrap(err, "thanos objstore secret yaml marshal failed")
	}

	thanosObjStoreSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "thanos-objstore-config",
		},
		StringData: map[string]string{
			"thanos.yaml":  string(secret),
			"objstore.yml": string(secret),
		},
	}

	k8sClient, err := k8s.NewFromFile(p.kubeconfigPath, logger)
	if err != nil {
		return errors.Wrap(err, "failed to set up the k8s client")
	}

	_, err = k8sClient.CreateOrUpdateNamespace(prometheus.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to create the prometheus namespace")
	}

	_, err = k8sClient.CreateOrUpdateSecret(prometheus.Namespace, thanosObjStoreSecret)
	if err != nil {
		return errors.Wrapf(err, "failed to create the Thanos object storage secret")
	}

	privateDomainName, err := p.awsClient.GetPrivateZoneDomainName(logger)
	if err != nil {
		return errors.Wrap(err, "unable to lookup private zone name")
	}

	app := "prometheus"
	dns := fmt.Sprintf("%s.%s.%s", p.cluster.ID, app, privateDomainName)

	h := p.newHelmDeployment(dns)

	err = h.Update()
	if err != nil {
		return errors.Wrap(err, "failed to create the Prometheus Operator Helm deployment")
	}

	err = p.updateVersion(h)
	if err != nil {
		return err
	}

	if p.awsClient.IsProvisionedPrivateCNAME(dns, p.logger) {
		p.logger.Debugln("CNAME was already provisioned for prometheus")
		return nil
	}

	p.logger.Debugln("CNAME was not provisioned for prometheus")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(180)*time.Second)
	defer cancel()

	endpoint, err := getPrivateLoadBalancerEndpoint(ctx, namespaceNginxInternal, logger.WithField("prometheus-action", "create"), p.kubeconfigPath)
	if err != nil {
		return errors.Wrap(err, "couldn't get the load balancer endpoint (nginx) for Prometheus")
	}

	logger.Infof("Registering DNS %s for %s", dns, app)
	err = p.awsClient.CreatePrivateCNAME(dns, []string{endpoint}, logger.WithField("prometheus-dns-create", dns))
	if err != nil {
		return errors.Wrap(err, "failed to create a CNAME to point to Prometheus")
	}

	return nil
}

func (p *prometheusOperator) ValuesPath() string {
	if p.desiredVersion == nil {
		return ""
	}
	return p.desiredVersion.Values()
}

func (p *prometheusOperator) Destroy() error {
	logger := p.logger.WithField("prometheus-action", "destroy")

	privateDomainName, err := p.awsClient.GetPrivateZoneDomainName(logger)
	if err != nil {
		return errors.Wrap(err, "unable to lookup private zone name")
	}
	app := "prometheus"
	dns := fmt.Sprintf("%s.%s.%s", p.cluster.ID, app, privateDomainName)

	logger.Infof("Deleting Route53 DNS Record for %s", app)
	err = p.awsClient.DeletePrivateCNAME(dns, logger.WithField("prometheus-dns-delete", dns))
	if err != nil {
		return errors.Wrap(err, "failed to delete Route53 DNS record")
	}

	p.actualVersion = nil

	helm := p.newHelmDeployment(dns)
	return helm.Delete()
}

func (p *prometheusOperator) Migrate() error {
	return nil
}

func (p *prometheusOperator) newHelmDeployment(prometheusDNS string) *helmDeployment {
	helmValueArguments := fmt.Sprintf("prometheus.prometheusSpec.externalLabels.clusterID=%s,prometheus.ingress.hosts={%s},prometheus.ingress.annotations.nginx\\.ingress\\.kubernetes\\.io/whitelist-source-range=%s", p.cluster.ID, prometheusDNS, strings.Join(p.allowCIDRRangeList, "\\,"))

	return newHelmDeployment(
		"prometheus-community/kube-prometheus-stack",
		"prometheus-operator",
		prometheus.Namespace,
		p.kubeconfigPath,
		p.desiredVersion,
		helmValueArguments,
		p.logger,
	)
}

func (p *prometheusOperator) Name() string {
	return model.PrometheusOperatorCanonicalName
}

func (p *prometheusOperator) DesiredVersion() *model.HelmUtilityVersion {
	return p.desiredVersion
}

func (p *prometheusOperator) ActualVersion() *model.HelmUtilityVersion {
	if p.actualVersion == nil {
		return nil
	}

	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(p.actualVersion.Version(), "kube-prometheus-stack-"),
		ValuesPath: p.actualVersion.Values(),
	}
}

func (p *prometheusOperator) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	p.actualVersion = actualVersion
	return nil
}
