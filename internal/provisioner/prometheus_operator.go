// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type prometheusOperator struct {
	awsClient      aws.AWS
	cluster        *model.Cluster
	kops           *kops.Cmd
	logger         log.FieldLogger
	provisioner    *KopsProvisioner
	desiredVersion string
	actualVersion  string
}

func newPrometheusOperatorHandle(cluster *model.Cluster, provisioner *KopsProvisioner, awsClient aws.AWS, kops *kops.Cmd, logger log.FieldLogger) (*prometheusOperator, error) {
	if logger == nil {
		return nil, fmt.Errorf("cannot instantiate Prometheus Operator handle with nil logger")
	}

	if cluster == nil {
		return nil, errors.New("cannot create a connection to Prometheus Operator if the cluster provided is nil")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to Prometheus Operator if the provisioner provided is nil")
	}

	if awsClient == nil {
		return nil, errors.New("cannot create a connection to Prometheus Operator if the awsClient provided is nil")
	}

	if kops == nil {
		return nil, errors.New("cannot create a connection to Prometheus Operator if the Kops command provided is nil")
	}

	version, err := cluster.DesiredUtilityVersion(model.PrometheusOperatorCanonicalName)
	if err != nil {
		return nil, errors.Wrap(err, "something went wrong while getting chart version for Prometheus Operator")
	}

	return &prometheusOperator{
		awsClient:      awsClient,
		cluster:        cluster,
		kops:           kops,
		logger:         logger.WithField("cluster-utility", model.PrometheusOperatorCanonicalName),
		provisioner:    provisioner,
		desiredVersion: version,
	}, nil
}

func (p *prometheusOperator) CreateOrUpgrade() error {
	logger := p.logger.WithField("prometheus-action", "create")

	h := p.NewHelmDeployment()
	err := h.TryMigrate()
	if err != nil {
		return errors.Wrap(err, "failed to migrate prometheus-operator release")
	}

	environment, err := p.awsClient.GetCloudEnvironmentName()
	if err != nil {
		return errors.Wrap(err, "failed to get environment name for thanos objstore secret")
	}

	if environment == "" {
		return errors.New("cannot create a thanos objstore secret if environment is empty")
	}

	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		awsRegion = aws.DefaultAWSRegion
	}

	secretData := map[string]interface{}{
		"type": "s3",
		"config": map[string]string{
			"bucket":   fmt.Sprintf("cloud-%s-prometheus-metrics", environment),
			"endpoint": fmt.Sprintf("s3.%s.amazonaws.com", awsRegion),
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
			"thanos.yaml": string(secret),
		},
	}

	k8sClient, err := k8s.NewFromFile(p.kops.GetKubeConfigPath(), logger)
	if err != nil {
		return errors.Wrap(err, "failed to set up the k8s client")
	}

	_, err = k8sClient.CreateOrUpdateNamespace("prometheus")
	if err != nil {
		return errors.Wrapf(err, "failed to create the prometheus namespace")
	}

	_, err = k8sClient.CreateOrUpdateSecret("prometheus", thanosObjStoreSecret)
	if err != nil {
		return errors.Wrapf(err, "failed to create the Thanos object storage secret")
	}

	err = h.Update()
	if err != nil {
		return errors.Wrap(err, "failed to create the Prometheus Operator Helm deployment")
	}

	err = p.updateVersion(h)
	if err != nil {
		return err
	}

	privateDomainName, err := p.awsClient.GetPrivateZoneDomainName(logger)
	if err != nil {
		return errors.Wrap(err, "unable to lookup private zone name")
	}

	app := "prometheus"
	dns := fmt.Sprintf("%s.%s.%s", p.cluster.ID, app, privateDomainName)
	if p.awsClient.IsProvisionedPrivateCNAME(dns, p.logger) {
		p.logger.Debugln("CNAME was already provisioned for prometheus")
		return nil
	}

	p.logger.Debugln("CNAME was not provisioned for prometheus")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(120)*time.Second)
	defer cancel()

	endpoint, err := getPrivateLoadBalancerEndpoint(ctx, "nginx", logger.WithField("prometheus-action", "create"), p.kops.GetKubeConfigPath())
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

	p.actualVersion = ""
	return nil
}

func (p *prometheusOperator) Migrate() error {
	return nil
}

func (p *prometheusOperator) NewHelmDeployment() *helmDeployment {
	privateDomainName, err := p.awsClient.GetPrivateZoneDomainName(p.logger)
	if err != nil {
		p.logger.WithError(err).Error("unable to lookup private zone name")
	}
	prometheusDNS := fmt.Sprintf("%s.prometheus.%s", p.cluster.ID, privateDomainName)

	helmValueArguments := fmt.Sprintf("prometheus.prometheusSpec.externalLabels.clusterID=%s,prometheus.ingress.hosts={%s},prometheus.ingress.annotations.nginx\\.ingress\\.kubernetes\\.io/whitelist-source-range=%s", p.cluster.ID, prometheusDNS, strings.Join(p.provisioner.allowCIDRRangeList, "\\,"))

	return &helmDeployment{
		chartDeploymentName: "prometheus-operator",
		chartName:           "prometheus-community/kube-prometheus-stack",
		kops:                p.kops,
		kopsProvisioner:     p.provisioner,
		logger:              p.logger,
		namespace:           "prometheus",
		setArgument:         helmValueArguments,
		valuesPath:          "helm-charts/prometheus_operator_values.yaml",
		desiredVersion:      p.desiredVersion,
	}
}

func (p *prometheusOperator) Name() string {
	return model.PrometheusOperatorCanonicalName
}

func (p *prometheusOperator) DesiredVersion() string {
	return p.desiredVersion
}

func (p *prometheusOperator) ActualVersion() string {
	return strings.TrimPrefix(p.actualVersion, "kube-prometheus-stack-")
}

func (p *prometheusOperator) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	p.actualVersion = actualVersion
	return nil
}
