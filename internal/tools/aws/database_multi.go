package aws

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"
	gt "github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"

	mysqlerr "github.com/go-mysql/errors"
	_ "github.com/go-sql-driver/mysql"
	"github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
)

const emptyString = ""

// RDSMultiDatabase is a database backed by AWS RDS that supports multi tenancy.
type RDSMultiDatabase struct {
	client         *Client
	installationID string
}

// NewRDSMultiDatabase returns a new RDSDatabase interface.
func NewRDSMultiDatabase(installationID string, client *Client) *RDSMultiDatabase {
	return &RDSMultiDatabase{
		client:         client,
		installationID: installationID,
	}
}

// Provision completes all the steps necessary to provision a multi-tenant RDS database.
func (d *RDSMultiDatabase) Provision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	// Steps:
	//  1. Check if database already exists by checking for the tagged secret tag:rds-database-cloud-id:cloud-id
	//	   If find it, check tag:rds-db-cluster-id:db-cluster-id
	//  2. Find all DB clusters tagged with tag:multi-tenant-rds-cluster:available
	//  3. Find the cluster with the least amount of databases.
	//  4. Provision database and tag the secret with the  tag:multi-tenant-rds-cluster:availableand tag:installation-id:id.
	//	   This will be used for fast lookup when we need to know in which cluster the database is installed.
	connString, err := d.getConnectionString()
	if err != nil {
		return err
	}

	// We should check if database really exists. If it does, we are done here.
	logger.Debugf("conn string: %s", connString)
	db, err := sql.Open("mysql", connString)
	if err != nil {
		return err
	}

	_, err = db.Query(fmt.Sprintf("SHOW DATABASES LIKE '%s'", d.installationID))
	if err != nil && mysqlerr.MySQLErrorCode(err) != 1049 {
		return err
	}

	logger.Debugf("database %s does not exist, finding cluster and creating one", d.installationID)
	return nil
}

// Teardown removes all AWS resources related to a multi-tenant RDS database.
func (d *RDSMultiDatabase) Teardown(keepData bool, logger log.FieldLogger) error {
	return nil
}

// Snapshot creates a snapshot of the multi-tenant multi-tenant RDS database..
func (d *RDSMultiDatabase) Snapshot(logger log.FieldLogger) error {
	return nil
}

// GenerateDatabaseSpecAndSecret creates the k8s database spec and secret for
// accessing the multi-tenant RDS database..
func (d *RDSMultiDatabase) GenerateDatabaseSpecAndSecret(logger log.FieldLogger) (*mmv1alpha1.Database, *corev1.Secret, error) {
	return nil, nil, nil
}

func (d *RDSMultiDatabase) getConnectionString() (string, error) {
	resourceNames, err := d.client.resourceTaggingGetAllResources(gt.GetResourcesInput{
		TagFilters: []*gt.TagFilter{
			{
				Key:    aws.String("rds-database-cloud-id"),
				Values: []*string{aws.String(CloudID(d.installationID))},
			},
			{
				Key: aws.String("rds-db-cluster-id"),
			},
		},
	})
	if err != nil {
		return emptyString, err
	}

	if len(resourceNames) < 1 {
		return emptyString, nil
	}

	for _, resource := range resourceNames {
		secret, err := d.client.Service().secretsManager.DescribeSecret(&secretsmanager.DescribeSecretInput{
			SecretId: resource.ResourceARN,
		})
		if err != nil {
			return emptyString, err
		}
		if secret.DeletedDate == nil {
			result, err := d.client.Service().secretsManager.GetSecretValue(&secretsmanager.GetSecretValueInput{
				SecretId: secret.ARN,
			})
			if err != nil {
				return emptyString, err
			}

			var rdsSecret *RDSSecret
			err = json.Unmarshal([]byte(*result.SecretString), &rdsSecret)
			if err != nil {
				return "", errors.Wrap(err, "unable to marshal secrets manager payload")
			}

			err = rdsSecret.Validate()
			if err != nil {
				return emptyString, err
			}

			for _, tag := range secret.Tags {
				if *tag.Key == "rds-db-cluster-id" {
					out, err := d.client.Service().rds.DescribeDBClusters(&rds.DescribeDBClustersInput{
						DBClusterIdentifier: tag.Value,
					})
					if err != nil {
						return emptyString, err
					}
					if len(out.DBClusters) != 1 {
						return emptyString, fmt.Errorf("expected 1 DB cluster, but got %d", len(out.DBClusters))
					}

					return fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4,utf8&readTimeout=30s&writeTimeout=30s",
						rdsSecret.MasterUsername, rdsSecret.MasterPassword, *out.DBClusters[0].Endpoint, d.installationID), nil
				}
			}
		}

	}

	return emptyString, nil
}
