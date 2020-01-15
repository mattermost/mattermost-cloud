package provisioner

import (
	"fmt"

	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	log "github.com/sirupsen/logrus"
)

type nginx struct {
	provisioner *KopsProvisioner
	kops        *kops.Cmd
	logger      log.FieldLogger
}

func newNginxHandle(provisioner *KopsProvisioner, kops *kops.Cmd, logger log.FieldLogger) (*nginx, error) {
	if logger == nil {
		return nil, fmt.Errorf("cannot instantiate NGINX handle with nil logger")
	}

	if provisioner == nil {
		return nil, fmt.Errorf("Cannot create a connection to Nginx if the provisioner provided is nil")
	}

	if kops == nil {
		return nil, fmt.Errorf("Cannot create a connection to Nginx if the Kops command provided is nil")
	}

	return &nginx{
		provisioner: provisioner,
		kops:        kops,
		logger:      logger.WithField("cluster-utility", "nginx"),
	}, nil

}

func (n *nginx) Create() error {
	return (&helmDeployment{
		chartDeploymentName: "private-nginx",
		chartName:           "stable/nginx-ingress",
		namespace:           "internal-nginx",
		setArgument:         "",
		valuesPath:          "helm-charts/private-nginx_values.yaml",
		kopsProvisioner:     n.provisioner,
		kops:                n.kops,
		logger:              n.logger,
	}).Create()
}

func (n *nginx) Upgrade() error {
	return nil
}

func (n *nginx) Destroy() error {
	return nil
}
