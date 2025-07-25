// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package pgbouncer

import (
	"context"
	"fmt"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const pgbouncerConfigmapDataKey = "pgbouncer.ini"

const baseIni = `
[pgbouncer]
listen_addr = *
listen_port = 5432
auth_file = /etc/userlist/userlist.txt
auth_query = SELECT usename, passwd FROM pgbouncer.pgbouncer_users WHERE usename=$1
auth_type = scram-sha-256
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
server_reset_query_always = %d

[databases]
`

func generatePGBouncerIni(vpcID string, store model.ClusterUtilityDatabaseStoreInterface, config *model.PgBouncerConfig) (string, error) {
	ini := generatePGBouncerBaseIni(config)

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

func GeneratePGBouncerUserlist(vpcID string, awsClient aws.AWS) (string, error) {
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

func generatePGBouncerBaseIni(c *model.PgBouncerConfig) string {
	return fmt.Sprintf(
		baseIni,
		c.MinPoolSize, c.DefaultPoolSize, c.ReservePoolSize,
		c.MaxClientConnections, c.MaxDatabaseConnectionsPerPool,
		c.ServerIdleTimeout, c.ServerLifetime, c.ServerResetQueryAlways,
	)
}

func UpdatePGBouncerConfigMap(ctx context.Context, vpc string, store model.ClusterUtilityDatabaseStoreInterface, pgbouncerConfig *model.PgBouncerConfig, k8sClient *k8s.KubeClient, logger log.FieldLogger) error {
	ini, err := generatePGBouncerIni(vpc, store, pgbouncerConfig)
	if err != nil {
		return errors.Wrap(err, "failed to generate updated pgbouncer ini contents")
	}

	configMap, err := k8sClient.Clientset.CoreV1().ConfigMaps("pgbouncer").Get(ctx, "pgbouncer-configmap", metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get configmap for pgbouncer-configmap")
	}
	if _, ok := configMap.Data[pgbouncerConfigmapDataKey]; !ok {
		logger.Warnf("No configmap key %s found for pgbouncer configmap; setting key with blank value", pgbouncerConfigmapDataKey)
		if configMap.Data == nil {
			configMap.Data = make(map[string]string)
		}
		configMap.Data[pgbouncerConfigmapDataKey] = ""
	}
	if configMap.Data[pgbouncerConfigmapDataKey] != ini {
		logger.Debug("Updating pgbouncer.ini with new database configuration")

		configMap.Data[pgbouncerConfigmapDataKey] = ini
		_, err = k8sClient.Clientset.CoreV1().ConfigMaps("pgbouncer").Update(ctx, configMap, metav1.UpdateOptions{})
		if err != nil {
			return errors.Wrap(err, "failed to update configmap pgbouncer-configmap")
		}
	}
	return nil
}
