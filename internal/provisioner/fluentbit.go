package provisioner

import (
	"fmt"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type fluentbit struct {
	provisioner *KopsProvisioner
	awsClient   aws.AWS
	kops        *kops.Cmd
	logger      log.FieldLogger
}

func newFluentbitHandle(provisioner *KopsProvisioner, awsClient aws.AWS, kops *kops.Cmd, logger log.FieldLogger) (*fluentbit, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate Fluentbit handle with nil logger")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to Fluentbit if the provisioner provided is nil")
	}

	if awsClient == nil {
		return nil, errors.New("cannot create a connection to Fluentbit if the awsClient provided is nil")
	}

	if kops == nil {
		return nil, errors.New("cannot create a connection to Fluentbit if the Kops command provided is nil")
	}

	return &fluentbit{
		provisioner: provisioner,
		awsClient:   awsClient,
		kops:        kops,
		logger:      logger.WithField("cluster-utility", "fluentbit"),
	}, nil
}

func (f *fluentbit) Create() error {
	logger := f.logger.WithField("fluentbit-action", "create")

	privateDomainName, err := f.awsClient.GetPrivateZoneDomainName(logger)
	if err != nil {
		logger.WithError(err).Error("unable to lookup private zone name")
	}
	elasticSearchDNS := fmt.Sprintf("elasticsearch.%s", privateDomainName)

	return (&helmDeployment{
		chartDeploymentName: "fluent-bit",
		chartName:           "stable/fluent-bit",
		namespace:           "fluent-bit",
		setArgument:         fmt.Sprintf("backend.es.host=%s", elasticSearchDNS),
		valuesPath:          "helm-charts/fluent-bit_values.yaml",
		kopsProvisioner:     f.provisioner,
		kops:                f.kops,
		logger:              f.logger,
	}).Create()
}

func (f *fluentbit) Destroy() error {
	return nil
}

func (f *fluentbit) Upgrade() error {
	return nil
}
