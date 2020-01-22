package provisioner

import (
	"context"
	"fmt"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const PROMETHEUS_CHART_NAME = "stable/prometheus"

type prometheus struct {
	awsClient   aws.AWS
	cluster     *model.Cluster
	kops        *kops.Cmd
	logger      log.FieldLogger
	provisioner *KopsProvisioner
	version     string
}

func newPrometheusHandle(cluster *model.Cluster, provisioner *KopsProvisioner, awsClient aws.AWS, kops *kops.Cmd, logger log.FieldLogger) (*prometheus, error) {
	if logger == nil {
		return nil, fmt.Errorf("cannot instantiate Prometheus handle with nil logger")
	}

	if cluster == nil {
		return nil, fmt.Errorf("Cannot create a connection to Prometheus if the cluster provided is nil")
	}

	if provisioner == nil {
		return nil, fmt.Errorf("Cannot create a connection to Prometheus if the provisioner provided is nil")
	}

	if awsClient == nil {
		return nil, fmt.Errorf("Cannot create a connection to Prometheus if the awsClient provided is nil")
	}

	if kops == nil {
		return nil, fmt.Errorf("Cannot create a connection to Prometheus if the Kops command provided is nil")
	}

	return &prometheus{
		awsClient:   awsClient,
		cluster:     cluster,
		kops:        kops,
		logger:      logger.WithField("cluster-utility", "prometheus"),
		provisioner: provisioner,
		version:     cluster.PrometheusVersion,
	}, nil
}

func (p *prometheus) Create() error {
	err := p.NewHelmDeployment().Create()
	if err != nil {
		return errors.Wrap(err, "failed to create the Prometheus Helm deployment")
	}

	logger := p.logger.WithField("prometheus-action", "create")

	app := "prometheus"
	dns := fmt.Sprintf("%s.%s.%s", p.cluster.ID, app, p.provisioner.privateDNS)

	ctx, cancel := context.WithTimeout(context.Background(), 120)
	defer cancel()

	endpoint, err := getLoadBalancerEndpoint(ctx, "internal-nginx", logger.WithField("prometheus-action", "create"), p.kops.GetKubeConfigPath())
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
	app := "prometheus"
	logger := p.logger.WithField("prometheus-action", "destroy")
	logger.Infof("Deleting Route53 DNS Record for %s", app)
	dns := fmt.Sprintf("%s.%s.%s", p.cluster.ID, app, p.provisioner.privateDNS)
	err := p.awsClient.DeletePrivateCNAME(dns, logger.WithField("prometheus-dns-delete", dns))
	if err != nil {
		return errors.Wrap(err, "failed to delete Route53 DNS record")
	}

	return nil
}

func (p *prometheus) Upgrade() error {
	return p.NewHelmDeployment().Update()
}

func (p *prometheus) NewHelmDeployment() *helmDeployment {
	prometheusDNS := fmt.Sprintf("%s.prometheus.%s", p.cluster.ID, p.provisioner.privateDNS)
	return &helmDeployment{
		chartDeploymentName: "prometheus",
		chartName:           PROMETHEUS_CHART_NAME,
		kops:                p.kops,
		kopsProvisioner:     p.provisioner,
		logger:              p.logger,
		namespace:           "prometheus",
		setArgument:         fmt.Sprintf("server.ingress.hosts={%s}", prometheusDNS),
		valuesPath:          "helm-charts/prometheus_values.yaml",
		version:             p.version,
	}
}
