// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/k8s"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type pgbouncer struct {
	awsClient      aws.AWS
	environment    string
	provisioner    *KopsProvisioner
	kops           *kops.Cmd
	cluster        *model.Cluster
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func newPgbouncerHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, provisioner *KopsProvisioner, awsClient aws.AWS, kops *kops.Cmd, logger log.FieldLogger) (*pgbouncer, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate Pgbouncer handle with nil logger")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to Pgbouncer if the provisioner provided is nil")
	}

	if kops == nil {
		return nil, errors.New("cannot create a connection to Pgbouncer if the Kops command provided is nil")
	}

	return &pgbouncer{
		awsClient:      awsClient,
		environment:    awsClient.GetCloudEnvironmentName(),
		provisioner:    provisioner,
		kops:           kops,
		cluster:        cluster,
		logger:         logger.WithField("cluster-utility", model.PgbouncerCanonicalName),
		desiredVersion: desiredVersion,
	}, nil

}

func (p *pgbouncer) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	p.actualVersion = actualVersion
	return nil
}

func (p *pgbouncer) ValuesPath() string {
	if p.desiredVersion == nil {
		return ""
	}
	return p.desiredVersion.Values()
}

func (p *pgbouncer) CreateOrUpgrade() error {
	err := p.DeployManifests()
	if err != nil {
		return err
	}

	h := p.NewHelmDeployment()

	err = h.Update()
	if err != nil {
		return err
	}

	err = p.updateVersion(h)
	return err
}

func (p *pgbouncer) DesiredVersion() *model.HelmUtilityVersion {
	return p.desiredVersion
}

func (p *pgbouncer) ActualVersion() *model.HelmUtilityVersion {
	if p.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(p.actualVersion.Version(), "pgbouncer-"),
		ValuesPath: p.actualVersion.Values(),
	}
}

func (p *pgbouncer) Destroy() error {
	return nil
}

func (p *pgbouncer) Migrate() error {
	return nil
}

func (p *pgbouncer) NewHelmDeployment() *helmDeployment {
	return &helmDeployment{
		chartDeploymentName: "pgbouncer",
		chartName:           "chartmuseum/pgbouncer",
		namespace:           "pgbouncer",
		kopsProvisioner:     p.provisioner,
		kops:                p.kops,
		logger:              p.logger,
		desiredVersion:      p.desiredVersion,
	}
}

// Deploys pgbouncer manifests if they don't exist: pgbouncer-configmap and pgbouncer-userlist-secret
func (p *pgbouncer) DeployManifests() error {
	logger := p.logger.WithField("pgbouncer-action", "create-manifests")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(180)*time.Second)
	defer cancel()

	k8sClient, err := k8s.NewFromFile(p.kops.GetKubeConfigPath(), logger)
	if err != nil {
		return errors.Wrap(err, "failed to set up the k8s client")
	}

	_, err = k8sClient.CreateOrUpdateNamespace("pgbouncer")
	if err != nil {
		return errors.Wrapf(err, "failed to create the pgbouncer namespace")
	}

	file := k8s.ManifestFile{}
	_, err = k8sClient.Clientset.CoreV1().ConfigMaps("pgbouncer").Get(ctx, "pgbouncer-configmap", metav1.GetOptions{})
	if k8sErrors.IsNotFound(err) {
		logger.Info("Configmap resource for pgbouncer-configmap does not exist, will be created...")
		file = k8s.ManifestFile{
			Path:            "manifests/pgbouncer-manifests/pgbouncer-configmap.yaml",
			DeployNamespace: "pgbouncer",
		}
		err = k8sClient.CreateFromFile(file, "")
		if err != nil {
			return err
		}
	} else if err != nil {
		return errors.Wrap(err, "failed to get configmap for pgbouncer-configmap")
	}

	_, err = k8sClient.Clientset.CoreV1().Secrets("pgbouncer").Get(ctx, "pgbouncer-userlist-secret", metav1.GetOptions{})
	if k8sErrors.IsNotFound(err) {
		logger.Info("Secret resource for pgbouncer-userlist-secret does not exist, will be created...")
		file = k8s.ManifestFile{
			Path:            "manifests/pgbouncer-manifests/pgbouncer-secret.yaml",
			DeployNamespace: "pgbouncer",
		}
		err = k8sClient.CreateFromFile(file, "")
		if err != nil {
			return err
		}
	} else if err != nil {
		return errors.Wrap(err, "failed to get secret for pgbouncer-userlist-secret")
	}
	return nil
}

func (p *pgbouncer) Name() string {
	return model.PgbouncerCanonicalName
}
