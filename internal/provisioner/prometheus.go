// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type prometheus struct {
	awsClient      aws.AWS
	cluster        *model.Cluster
	kops           *kops.Cmd
	logger         log.FieldLogger
	provisioner    *KopsProvisioner
	desiredVersion string
	actualVersion  string
}

func newPrometheusHandle(cluster *model.Cluster, provisioner *KopsProvisioner, awsClient aws.AWS, kops *kops.Cmd, logger log.FieldLogger) (*prometheus, error) {
	if logger == nil {
		return nil, fmt.Errorf("cannot instantiate Prometheus handle with nil logger")
	}

	if cluster == nil {
		return nil, errors.New("cannot create a connection to Prometheus if the cluster provided is nil")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to Prometheus if the provisioner provided is nil")
	}

	if awsClient == nil {
		return nil, errors.New("cannot create a connection to Prometheus if the awsClient provided is nil")
	}

	if kops == nil {
		return nil, errors.New("cannot create a connection to Prometheus if the Kops command provided is nil")
	}

	version, err := cluster.DesiredUtilityVersion(model.PrometheusCanonicalName)
	if err != nil {
		return nil, errors.Wrap(err, "something went wrong while getting chart version for Prometheus")
	}

	return &prometheus{
		awsClient:      awsClient,
		cluster:        cluster,
		kops:           kops,
		logger:         logger.WithField("cluster-utility", model.PrometheusCanonicalName),
		provisioner:    provisioner,
		desiredVersion: version,
	}, nil
}

func (p *prometheus) CreateOrUpgrade() error {
	logger := p.logger.WithField("prometheus-action", "create")
	h := p.NewHelmDeployment()
	err := h.Update()
	if err != nil {
		return errors.Wrap(err, "failed to create the Prometheus Helm deployment")
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
	ctx, cancel := context.WithTimeout(context.Background(), 120)
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

func (p *prometheus) Destroy() error {
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

func (p *prometheus) NewHelmDeployment() *helmDeployment {
	privateDomainName, err := p.awsClient.GetPrivateZoneDomainName(p.logger)
	if err != nil {
		p.logger.WithError(err).Error("unable to lookup private zone name")
	}
	prometheusDNS := fmt.Sprintf("%s.prometheus.%s", p.cluster.ID, privateDomainName)

	helmValueArguments := fmt.Sprintf("server.ingress.hosts={%[1]s},server.ingress.annotations.nginx\\.ingress\\.kubernetes\\.io/configuration-snippet='if ($host != \"%[1]s\") {return 404;}'", prometheusDNS)

	return &helmDeployment{
		chartDeploymentName: "prometheus",
		chartName:           "stable/prometheus",
		kops:                p.kops,
		kopsProvisioner:     p.provisioner,
		logger:              p.logger,
		namespace:           "prometheus",
		setArgument:         helmValueArguments,
		valuesPath:          "helm-charts/prometheus_values.yaml",
		desiredVersion:      p.desiredVersion,
	}
}

func (p *prometheus) Name() string {
	return model.PrometheusCanonicalName
}

func (p *prometheus) DesiredVersion() string {
	return p.desiredVersion
}

func (p *prometheus) ActualVersion() string {
	return strings.TrimPrefix(p.actualVersion, "prometheus-")
}

func (p *prometheus) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	p.actualVersion = actualVersion
	return nil
}
