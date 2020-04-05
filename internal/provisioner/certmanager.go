package provisioner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/k8s"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/internal/tools/utils"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type certManager struct {
	provisioner    *KopsProvisioner
	kops           *kops.Cmd
	logger         log.FieldLogger
	desiredVersion string
	actualVersion  string
}

func newCertManagerHandle(desiredVersion string, provisioner *KopsProvisioner, kops *kops.Cmd, logger log.FieldLogger) (*certManager, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate Cert Manager handle with nil logger")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to Cert Manager if the provisioner provided is nil")
	}

	if kops == nil {
		return nil, errors.New("cannot create a connection to Cert Manager if the Kops command provided is nil")
	}

	return &certManager{
		provisioner:    provisioner,
		kops:           kops,
		logger:         logger.WithField("cluster-utility", model.CertManagerCanonicalName),
		desiredVersion: desiredVersion,
	}, nil

}

func (n *certManager) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	n.actualVersion = actualVersion
	return nil
}

func (n *certManager) Create() error {
	h := n.NewHelmDeployment()
	err := h.Create()
	if err != nil {
		return err
	}

	err = n.updateVersion(h)
	return err
}

func (n *certManager) Upgrade() error {
	h := n.NewHelmDeployment()
	err := h.Update()
	if err != nil {
		return err
	}

	err = n.updateVersion(h)
	return err
}

func (n *certManager) DesiredVersion() string {
	return n.desiredVersion
}

func (n *certManager) ActualVersion() string {
	return strings.TrimPrefix(n.actualVersion, "cert-manager-")
}

func (n *certManager) Destroy() error {
	return nil
}

func (n *certManager) NewHelmDeployment() *helmDeployment {
	return &helmDeployment{
		chartDeploymentName: "cert-manager",
		chartName:           "jetstack/cert-manager",
		namespace:           "cert-manager",
		setArgument:         "",
		valuesPath:          "helm-charts/cert-manager_values.yaml",
		kopsProvisioner:     n.provisioner,
		kops:                n.kops,
		logger:              n.logger,
		desiredVersion:      n.desiredVersion,
	}
}

func (n *certManager) Name() string {
	return model.CertManagerCanonicalName
}

func deployCertManagerCRDS(kops *kops.Cmd, logger log.FieldLogger) error {
	files := []k8s.ManifestFile{
		{
			Path:            "certmanager-manifests/crds.yaml",
			DeployNamespace: "cert-manager",
		},
	}
	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return err
	}
	logger.Infof("Deploying cert manager crds")
	err = k8sClient.CreateFromFiles(files)
	if err != nil {
		return err
	}
	logger.Infof("Successfully deployed cert manager crds")
	return nil
}

func deployClusterIssuer(kops *kops.Cmd, logger log.FieldLogger) error {
	k8sClient, err := k8s.New(kops.GetKubeConfigPath(), logger)
	if err != nil {
		return err
	}

	wait := 300
	pods, err := k8sClient.GetPodsFromDeployment("cert-manager", "cert-manager-webhook")
	if err != nil {
		return err
	}
	if len(pods.Items) == 0 {
		return fmt.Errorf("no pods found from cert-manager-webhook deployment")
	}

	for _, pod := range pods.Items {
		logger.Infof("Waiting up to %d seconds for cert-manager-webhook pod %q to start...", wait, pod.GetName())
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
		defer cancel()
		pod, err := k8sClient.WaitForPodRunning(ctx, "cert-manager", pod.GetName())
		if err != nil {
			return err
		}
		logger.Infof("Successfully deployed cert-manager-webhook pod %q", pod.Name)
	}

	maxRetries := 10
	logger.Infof("Trying up to %d times for cluster issuer to be deployed", maxRetries)
	err = utils.Retry(maxRetries, time.Second*2, func() error {
		files := []k8s.ManifestFile{
			{
				Path:            "certmanager-manifests/cluster-issuer.yaml",
				DeployNamespace: "cert-manager",
			},
		}

		err = k8sClient.CreateFromFiles(files)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
