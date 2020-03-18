package provisioner

import (
	"strings"

	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type publicNginx struct {
	provisioner    *KopsProvisioner
	kops           *kops.Cmd
	logger         log.FieldLogger
	desiredVersion string
	actualVersion  string
}

func newPublicNginxHandle(desiredVersion string, provisioner *KopsProvisioner, kops *kops.Cmd, logger log.FieldLogger) (*publicNginx, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate Public NGINX handle with nil logger")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to Public Nginx if the provisioner provided is nil")
	}

	if kops == nil {
		return nil, errors.New("cannot create a connection to Public Nginx if the Kops command provided is nil")
	}

	return &publicNginx{
		provisioner:    provisioner,
		kops:           kops,
		logger:         logger.WithField("cluster-utility", model.PublicNginxCanonicalName),
		desiredVersion: desiredVersion,
	}, nil

}

func (n *publicNginx) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	n.actualVersion = actualVersion
	return nil
}

func (n *publicNginx) Create() error {
	h := n.NewHelmDeployment()
	err := h.Create()
	if err != nil {
		return err
	}

	err = n.updateVersion(h)
	return err
}

func (n *publicNginx) Upgrade() error {
	h := n.NewHelmDeployment()
	err := h.Update()
	if err != nil {
		return err
	}

	err = n.updateVersion(h)
	return err
}

func (n *publicNginx) DesiredVersion() string {
	return n.desiredVersion
}

func (n *publicNginx) ActualVersion() string {
	return strings.TrimPrefix(n.actualVersion, "nginx-ingress-")
}

func (n *publicNginx) Destroy() error {
	return nil
}

func (n *publicNginx) NewHelmDeployment() *helmDeployment {
	return &helmDeployment{
		chartDeploymentName: "public-nginx",
		chartName:           "stable/nginx-ingress",
		namespace:           "public-nginx",
		setArgument:         "",
		valuesPath:          "helm-charts/public-nginx_values.yaml",
		kopsProvisioner:     n.provisioner,
		kops:                n.kops,
		logger:              n.logger,
		desiredVersion:      n.desiredVersion,
	}
}

func (n *publicNginx) Name() string {
	return model.PublicNginxCanonicalName
}
