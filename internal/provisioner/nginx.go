package provisioner

import (
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type nginx struct {
	provisioner    *KopsProvisioner
	kops           *kops.Cmd
	logger         log.FieldLogger
	desiredVersion string
	actualVersion  string
}

func newNginxHandle(desiredVersion string, provisioner *KopsProvisioner, kops *kops.Cmd, logger log.FieldLogger) (*nginx, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate NGINX handle with nil logger")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to Nginx if the provisioner provided is nil")
	}

	if kops == nil {
		return nil, errors.New("cannot create a connection to Nginx if the Kops command provided is nil")
	}

	return &nginx{
		provisioner:    provisioner,
		kops:           kops,
		logger:         logger.WithField("cluster-utility", model.NginxCanonicalName),
		desiredVersion: desiredVersion,
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
	h := n.NewHelmDeployment()
	err := h.Update()
	if err != nil {
		return err
	}

	err = n.updateVersion(h)
	return err
}

func (n *nginx) DesiredVersion() string {
	return n.desiredVersion
}

func (n *nginx) ActualVersion() string {
	return strings.TrimPrefix(n.actualVersion, "nginx-ingress-")
}

func (n *nginx) Destroy() error {
	return nil
}

func (n *nginx) NewHelmDeployment() *helmDeployment {
	return &helmDeployment{
		chartDeploymentName: "private-nginx",
		chartName:           "stable/nginx-ingress",
		namespace:           "internal-nginx",
		setArgument:         "",
		valuesPath:          "helm-charts/private-nginx_values.yaml",
		kopsProvisioner:     n.provisioner,
		kops:                n.kops,
		logger:              n.logger,
		desiredVersion:      n.desiredVersion,
	}
}

func (n *nginx) Name() string {
	return model.NginxCanonicalName
}
