package provisioner

import (
	"fmt"

	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	log "github.com/sirupsen/logrus"
)

type fluentbit struct {
	provisioner *KopsProvisioner
	kops        *kops.Cmd
	logger      log.FieldLogger
}

func newFluentbitHandle(provisioner *KopsProvisioner, kops *kops.Cmd, logger log.FieldLogger) (*fluentbit, error) {
	if logger == nil {
		return nil, fmt.Errorf("cannot instantiate Fluentbit handle with nil logger")
	}

	if provisioner == nil {
		return nil, fmt.Errorf("cannot create a connection to Fluentbit if the provisioner provided is nil")
	}

	if kops == nil {
		return nil, fmt.Errorf("cannot create a connection to Fluentbit if the Kops command provided is nil")
	}

	return &fluentbit{
		provisioner: provisioner,
		kops:        kops,
		logger:      logger.WithField("cluster-utility", "fluentbit"),
	}, nil
}

func (f *fluentbit) Create() error {
	elasticSearchDNS := fmt.Sprintf("elasticsearch.%s", f.provisioner.privateDNS)
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
