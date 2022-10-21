// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/k8s"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
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

func newPgbouncerHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, kubeconfigPath string, awsClient aws.AWS, logger log.FieldLogger) (*pgbouncer, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate Pgbouncer handle with nil logger")
	}
	if kubeconfigPath == "" {
		return nil, errors.New("cannot create utility without kubeconfig")
	}

	return &pgbouncer{
		awsClient:      awsClient,
		environment:    awsClient.GetCloudEnvironmentName(),
		cluster:        cluster,
		kubeconfigPath: kubeconfigPath,
		logger:         logger.WithField("cluster-utility", model.PgbouncerCanonicalName),
		desiredVersion: desiredVersion,
		actualVersion:  cluster.UtilityMetadata.ActualVersions.Pgbouncer,
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
		kubeconfigPath:      p.kubeconfigPath,
		logger:              p.logger,
		desiredVersion:      p.desiredVersion,
	}
}

// Deploys pgbouncer manifests if they don't exist: pgbouncer-configmap and pgbouncer-userlist-secret
func (p *pgbouncer) DeployManifests() error {
	logger := p.logger.WithField("pgbouncer-action", "create-manifests")

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(180)*time.Second)
	defer cancel()

	k8sClient, err := k8s.NewFromFile(p.kubeconfigPath, logger)
	if err != nil {
		return errors.Wrap(err, "failed to set up the k8s client")
	}

	_, err = k8sClient.CreateOrUpdateNamespace("pgbouncer")
	if err != nil {
		return errors.Wrapf(err, "failed to create the pgbouncer namespace")
	}

	// Both of these files should only be created on the first provision and
	// should never be overwritten with cluster provisioning afterwards.
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

const baseIni = `
[pgbouncer]
listen_addr = *
listen_port = 5432
auth_file = /etc/userlist/userlist.txt
auth_query = SELECT usename, passwd FROM pgbouncer.get_auth($1)
admin_users = admin
ignore_startup_parameters = extra_float_digits
tcp_keepalive = 1
tcp_keepcnt = 5
tcp_keepidle = 5
tcp_keepintvl = 1
server_round_robin = 1
log_disconnections = 1
log_connections = 1
pool_mode = transaction
min_pool_size = %d
default_pool_size = %d
reserve_pool_size = %d
reserve_pool_timeout = 1
max_client_conn = %d
max_db_connections = %d
server_idle_timeout = %d
server_lifetime = %d
server_reset_query_always = 1

[databases]
`

func generatePGBouncerIni(vpcID string, store model.ClusterUtilityDatabaseStoreInterface) (string, error) {
	ini := generatePGBouncerBaseIni()

	multitenantDatabases, err := store.GetMultitenantDatabases(&model.MultitenantDatabaseFilter{
		DatabaseType:          model.DatabaseEngineTypePostgresProxy,
		MaxInstallationsLimit: model.NoInstallationsLimit,
		Paging:                model.AllPagesNotDeleted(),
		VpcID:                 vpcID,
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to get proxy databases")
	}
	for _, multitenantDatabase := range multitenantDatabases {
		logicalDatabases, err := store.GetLogicalDatabases(&model.LogicalDatabaseFilter{
			MultitenantDatabaseID: multitenantDatabase.ID,
			Paging:                model.AllPagesNotDeleted(),
		})
		if err != nil {
			return "", errors.Wrap(err, "failed to get logical databases")
		}
		for _, logicalDatabase := range logicalDatabases {
			// Add writer entry.
			ini = fmt.Sprintf("%s%s = dbname=%s host=%s port=5432 auth_user=%s\n",
				ini,
				logicalDatabase.Name,
				logicalDatabase.Name,
				multitenantDatabase.WriterEndpoint,
				aws.DefaultPGBouncerAuthUsername,
			)

			// Add reader entry.
			ini = fmt.Sprintf("%s%s-ro = dbname=%s host=%s port=5432 auth_user=%s\n",
				ini,
				logicalDatabase.Name,
				logicalDatabase.Name,
				multitenantDatabase.ReaderEndpoint,
				aws.DefaultPGBouncerAuthUsername,
			)
		}
	}

	return ini, nil
}

func generatePGBouncerBaseIni() string {
	return fmt.Sprintf(baseIni, model.GetMinPoolSize(), model.GetDefaultPoolSize(), model.GetReservePoolSize(), model.GetMaxClientConnections(), model.GetMaxDatabaseConnectionsPerPool(), model.GetServerIdleTimeout(), model.GetServerLifetime())
}

func generatePGBouncerUserlist(vpcID string, awsClient aws.AWS) (string, error) {
	password, err := awsClient.SecretsManagerGetPGBouncerAuthUserPassword(vpcID)
	if err != nil {
		return "", errors.Wrap(err, "failed to get pgbouncer auth user password")
	}

	// WARNING: The admin account credentials must match what is deployed with
	// the helm chart values.
	userlist := fmt.Sprintf(
		"\"admin\" \"adminpassword\"\n\"%s\" \"%s\"\n",
		aws.DefaultPGBouncerAuthUsername,
		password,
	)

	return userlist, nil
}
