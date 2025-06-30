// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utility

import (
	"context"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type pgbouncer struct {
	awsClient      aws.AWS
	environment    string
	kubeconfigPath string
	cluster        *model.Cluster
	logger         log.FieldLogger
	desiredVersion *model.HelmUtilityVersion
	actualVersion  *model.HelmUtilityVersion
}

func newPgbouncerOrUnmanagedHandle(cluster *model.Cluster, kubeconfigPath string, awsClient aws.AWS, logger log.FieldLogger) (Utility, error) {
	desired := cluster.DesiredUtilityVersion(model.PgbouncerCanonicalName)
	actual := cluster.ActualUtilityVersion(model.PgbouncerCanonicalName)

	if model.UtilityIsUnmanaged(desired, actual) {
		return newUnmanagedHandle(model.PgbouncerCanonicalName, kubeconfigPath, []string{}, cluster, awsClient, logger), nil
	}

	pgbouncer := newPgbouncerHandle(cluster, desired, kubeconfigPath, awsClient, logger)
	err := pgbouncer.validate()
	if err != nil {
		return nil, errors.Wrap(err, "pgbouncer utility config is invalid")
	}

	return pgbouncer, nil
}

func newPgbouncerHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, kubeconfigPath string, awsClient aws.AWS, logger log.FieldLogger) *pgbouncer {
	return &pgbouncer{
		awsClient:      awsClient,
		environment:    awsClient.GetCloudEnvironmentName(),
		cluster:        cluster,
		kubeconfigPath: kubeconfigPath,
		logger:         logger.WithField("cluster-utility", model.PgbouncerCanonicalName),
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.Pgbouncer,
	}
}

func (p *pgbouncer) validate() error {
	if p.kubeconfigPath == "" {
		return errors.New("kubeconfig path cannot be empty")
	}

	return nil
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
	k8sClient, err := k8s.NewFromFile(p.kubeconfigPath, p.logger)
	if err != nil {
		return errors.Wrap(err, "failed to set up the k8s client")
	}

	err = DeployPgbouncerManifests(k8sClient, p.logger)
	if err != nil {
		return err
	}

	h := p.newHelmDeployment()

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
	helm := p.newHelmDeployment()
	return helm.Delete()
}

func (p *pgbouncer) Migrate() error {
	return nil
}

func (p *pgbouncer) newHelmDeployment() *helmDeployment {
	return newHelmDeployment(
		"chartmuseum/pgbouncer",
		"pgbouncer",
		"pgbouncer",
		p.kubeconfigPath,
		p.desiredVersion,
		defaultHelmDeploymentSetArgument,
		p.logger,
	)
}

func (p *pgbouncer) Name() string {
	return model.PgbouncerCanonicalName
}

// DeployManifests deploy pgbouncer manifests if they don't exist:
// pgbouncer-configmap and pgbouncer-userlist-secret
func DeployPgbouncerManifests(k8sClient *k8s.KubeClient, logger log.FieldLogger) error {
	logger = logger.WithField("pgbouncer-action", "create-manifests")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(180)*time.Second)
	defer cancel()

	_, err := k8sClient.CreateOrUpdateNamespace("pgbouncer")
	if err != nil {
		return errors.Wrapf(err, "failed to create the pgbouncer namespace")
	}

	// Both of these files should only be created on the first provision and
	// should never be overwritten with cluster provisioning afterwards.
	var file k8s.ManifestFile
	_, err = k8sClient.Clientset.CoreV1().ConfigMaps("pgbouncer").Get(ctx, "pgbouncer-configmap", metav1.GetOptions{})
	if k8sErrors.IsNotFound(err) {
		logger.Info("Configmap resource for pgbouncer-configmap does not exist, will be created...")
		file = k8s.ManifestFile{
			Path:            "manifests/pgbouncer-manifests/pgbouncer-configmap.yaml",
			DeployNamespace: "pgbouncer",
		}
		err = k8sClient.CreateFromFile(file, "")
		if err != nil {
			return errors.Wrap(err, "failed to create pgbouncer-configmap")
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
