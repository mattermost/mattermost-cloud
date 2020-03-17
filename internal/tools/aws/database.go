package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/pkg/apis/mattermost/v1alpha1"
)

const connStringTemplate = "mysql://%s:%s@tcp(%s:3306)/mattermost?charset=utf8mb4,utf8&readTimeout=30s&writeTimeout=30s"

// RDSDatabase is a database backed by AWS RDS.
type RDSDatabase struct {
	client         *Client
	installationID string
}

// NewRDSDatabase returns a new RDSDatabase interface.
func NewRDSDatabase(installationID string, client *Client) *RDSDatabase {
	return &RDSDatabase{
		client:         client,
		installationID: installationID,
	}
}

// Provision completes all the steps necessary to provision a RDS database.
func (d *RDSDatabase) Provision(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	d.client.AddSQLStore(store)

	err := d.rdsDatabaseProvision(d.installationID, logger)
	if err != nil {
		return errors.Wrap(err, "unable to provision RDS database")
	}

	return nil
}

// Teardown removes all AWS resources related to a RDS database.
func (d *RDSDatabase) Teardown(keepData bool, logger log.FieldLogger) error {
	awsID := CloudID(d.installationID)

	logger = logger.WithField("db-cluster-name", awsID)
	logger.Info("Tearing down AWS RDS database")

	err := d.client.secretsManagerEnsureRDSSecretDeleted(awsID, logger)
	if err != nil {
		return errors.Wrap(err, "unable to delete RDS secret")
	}

	if keepData {
		logger.Info("AWS RDS DB cluster was left intact due to the keep-data setting of this server")
		return nil
	}

	err = d.client.rdsEnsureDBClusterDeleted(awsID, logger)
	if err != nil {
		return errors.Wrap(err, "unable to delete RDS DB cluster")
	}

	logger.Debug("AWS RDS DB cluster deleted")
	return nil
}

// Snapshot creates a snapshot of the RDS database.
func (d *RDSDatabase) Snapshot(logger log.FieldLogger) error {
	dbClusterID := CloudID(d.installationID)

	_, err := d.client.Service(logger).rds.CreateDBClusterSnapshot(&rds.CreateDBClusterSnapshotInput{
		DBClusterIdentifier:         aws.String(dbClusterID),
		DBClusterSnapshotIdentifier: aws.String(fmt.Sprintf("%s-snapshot-%v", dbClusterID, time.Now().Nanosecond())),
		Tags: []*rds.Tag{&rds.Tag{
			Key:   aws.String(DefaultClusterInstallationSnapshotTagKey),
			Value: aws.String(RDSSnapshotTagValue(dbClusterID)),
		}},
	})
	if err != nil {
		return errors.Wrap(err, "failed to create a DB cluster snapshot")
	}

	logger.WithField("installation-id", d.installationID).Info("RDS database snapshot in progress")

	return nil
}

// GenerateDatabaseSpecAndSecret creates the k8s database spec and secret for
// accessing the RDS database.
func (d *RDSDatabase) GenerateDatabaseSpecAndSecret(logger log.FieldLogger) (*mmv1alpha1.Database, *corev1.Secret, error) {
	awsID := CloudID(d.installationID)

	rdsSecret, err := d.client.secretsManagerGetRDSSecret(awsID, logger)
	if err != nil {
		return nil, nil, err
	}

	result, err := d.client.Service(logger).rds.DescribeDBClusters(&rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(awsID),
	})
	if err != nil {
		return nil, nil, err
	}

	if len(result.DBClusters) != 1 {
		return nil, nil, fmt.Errorf("expected 1 DB cluster, but got %d", len(result.DBClusters))
	}

	databaseSecretName := fmt.Sprintf("%s-rds", d.installationID)

	databaseSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: databaseSecretName,
		},
		StringData: map[string]string{
			"DB_CONNECTION_STRING": fmt.Sprintf(connStringTemplate, rdsSecret.MasterUsername, rdsSecret.MasterPassword, *result.DBClusters[0].Endpoint),
		},
	}

	databaseSpec := &mmv1alpha1.Database{
		Secret: databaseSecretName,
	}

	logger.Debug("Cluster installation configured to use an AWS RDS Database")

	return databaseSpec, databaseSecret, nil
}

func (d *RDSDatabase) rdsDatabaseProvision(installationID string, logger log.FieldLogger) error {
	awsID := CloudID(installationID)
	logger.Infof("Provisioning AWS RDS database with ID %s", awsID)

	// To properly provision the database we need a SQL client to lookup which
	// cluster(s) the installation is running on.
	if !d.client.HasSQLStore() {
		return errors.New("the provided AWS client does not have SQL store access")
	}

	clusterInstallations, err := d.client.store.GetClusterInstallations(&model.ClusterInstallationFilter{
		PerPage:        model.AllPerPage,
		InstallationID: installationID,
	})
	if err != nil {
		return errors.Wrapf(err, "unable to lookup cluster installations for installation %s", installationID)
	}

	clusterInstallationCount := len(clusterInstallations)
	if clusterInstallationCount == 0 {
		return fmt.Errorf("no cluster installations found for %s", installationID)
	}
	if clusterInstallationCount != 1 {
		return fmt.Errorf("RDS provisioning is not currently supported for multiple cluster installations (found %d)", clusterInstallationCount)
	}

	clusterID := clusterInstallations[0].ClusterID
	vpcFilters := []*ec2.Filter{
		{
			Name:   aws.String(VpcClusterIDTagKey),
			Values: []*string{aws.String(clusterID)},
		},
		{
			Name:   aws.String(VpcAvailableTagKey),
			Values: []*string{aws.String(VpcAvailableTagValueFalse)},
		},
	}
	vpcs, err := d.client.GetVpcsWithFilters(vpcFilters, logger)
	if err != nil {
		return err
	}
	if len(vpcs) != 1 {
		return fmt.Errorf("expected 1 VPC for cluster %s, but got %d", clusterID, len(vpcs))
	}

	rdsSecret, err := d.client.secretsManagerEnsureRDSSecretCreated(awsID, logger)
	if err != nil {
		return err
	}

	err = d.client.rdsEnsureDBClusterCreated(awsID, *vpcs[0].VpcId, rdsSecret.MasterUsername, rdsSecret.MasterPassword, logger)
	if err != nil {
		return err
	}

	err = d.client.rdsEnsureDBClusterInstanceCreated(awsID, fmt.Sprintf("%s-master", awsID), logger)
	if err != nil {
		return err
	}

	return nil
}
