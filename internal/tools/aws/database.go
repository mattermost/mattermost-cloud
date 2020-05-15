package aws

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/aws/aws-sdk-go/service/resourcegroupstaggingapi"
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
func (d *RDSDatabase) Teardown(store model.InstallationDatabaseStoreInterface, keepData bool, logger log.FieldLogger) error {
	awsID := CloudID(d.installationID)

	logger = logger.WithField("db-cluster-name", awsID)
	logger.Info("Tearing down RDS DB cluster")

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

	resourceNames, err := d.getKMSResourceNames(awsID)
	if err != nil {
		return errors.Wrapf(err, "unabled to get KMS resources associated with db cluster %s", awsID)
	}

	if len(resourceNames) > 0 {
		enabledKeys, err := d.getEnabledEncryptionKeys(resourceNames)
		if err != nil {
			return errors.Wrapf(err, "unabled to get encryption key associated with db cluster %s", awsID)
		}

		for _, keyMetadata := range enabledKeys {
			err = d.client.kmsScheduleKeyDeletion(*keyMetadata.KeyId, KMSMaxTimeEncryptionKeyDeletion)
			if err != nil {
				return errors.Wrapf(err, "encryption key associated with db cluster %s could not be scheduled for deletion", awsID)
			}
			logger.Infof("Encryption key %s scheduled for deletion in %d days", *keyMetadata.Arn, KMSMaxTimeEncryptionKeyDeletion)
		}
	} else {
		logger.Warn("Could not find any encryption key. It has been already deleted or never created.")
	}

	logger.Debug("AWS RDS db cluster teardown completed")

	return nil
}

// Snapshot creates a snapshot of the RDS database.
func (d *RDSDatabase) Snapshot(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) error {
	dbClusterID := CloudID(d.installationID)

	_, err := d.client.Service().rds.CreateDBClusterSnapshot(&rds.CreateDBClusterSnapshotInput{
		DBClusterIdentifier:         aws.String(dbClusterID),
		DBClusterSnapshotIdentifier: aws.String(fmt.Sprintf("%s-snapshot-%v", dbClusterID, time.Now().Nanosecond())),
		Tags: []*rds.Tag{
			{
				Key:   aws.String(DefaultClusterInstallationSnapshotTagKey),
				Value: aws.String(RDSSnapshotTagValue(dbClusterID)),
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "failed to create a DB cluster snapshot")
	}

	logger.WithField("installation-id", d.installationID).Info("RDS database snapshot in progress")

	return nil
}

// GenerateDatabaseSpecAndSecret creates the k8s database spec and secret for
// accessing the RDS database.
func (d *RDSDatabase) GenerateDatabaseSpecAndSecret(store model.InstallationDatabaseStoreInterface, logger log.FieldLogger) (*mmv1alpha1.Database, *corev1.Secret, error) {
	awsID := CloudID(d.installationID)

	rdsSecret, err := d.client.secretsManagerGetRDSSecret(awsID, logger)
	if err != nil {
		return nil, nil, err
	}

	result, err := d.client.Service().rds.DescribeDBClusters(&rds.DescribeDBClustersInput{
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
	vpcs, err := d.client.GetVpcsWithFilters(vpcFilters)
	if err != nil {
		return err
	}
	if len(vpcs) != 1 {
		return fmt.Errorf("expected 1 VPC for cluster %s (found %d)", clusterID, len(vpcs))
	}

	rdsSecret, err := d.client.secretsManagerEnsureRDSSecretCreated(awsID, logger)
	if err != nil {
		return err
	}

	kmsResourceNames, err := d.getKMSResourceNames(awsID)
	if err != nil {
		return err
	}

	var keyMetadata *kms.KeyMetadata
	if len(kmsResourceNames) > 0 {
		enabledKeys, err := d.getEnabledEncryptionKeys(kmsResourceNames)
		if err != nil {
			return errors.Wrapf(err, "failed to get encryption keys for db cluster %s", awsID)
		}

		if len(enabledKeys) != 1 {
			return errors.Errorf("db cluster %s should have exactly one enabled/active encryption key (found %d)", awsID, len(enabledKeys))
		}

		keyMetadata = enabledKeys[0]
	} else {
		keyMetadata, err = d.client.kmsCreateSymmetricKey(KMSKeyDescriptionRDS(awsID), []*kms.Tag{
			{
				TagKey:   aws.String(DefaultRDSEncryptionTagKey),
				TagValue: aws.String(awsID),
			},
		})
		if err != nil {
			return errors.Wrapf(err, "unable to create an encryption key for db cluster %s", awsID)
		}
	}

	logger.Infof("Encrypting RDS database with key %s", *keyMetadata.Arn)

	err = d.client.rdsEnsureDBClusterCreated(awsID, *vpcs[0].VpcId, rdsSecret.MasterUsername, rdsSecret.MasterPassword, *keyMetadata.KeyId, logger)
	if err != nil {
		return err
	}

	err = d.client.rdsEnsureDBClusterInstanceCreated(awsID, fmt.Sprintf("%s-master", awsID), logger)
	if err != nil {
		return err
	}

	return nil
}

func (d *RDSDatabase) getKMSResourceNames(awsID string) ([]*string, error) {
	kmsResources, err := d.client.resourceTaggingGetAllResources(resourcegroupstaggingapi.GetResourcesInput{
		TagFilters: []*resourcegroupstaggingapi.TagFilter{
			{
				Key:    aws.String(DefaultRDSEncryptionTagKey),
				Values: []*string{aws.String(awsID)},
			},
		},
	})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get KMS resources with tag %s:%s", DefaultRDSEncryptionTagKey, awsID)
	}

	resourceNameList := make([]*string, len(kmsResources))
	for i, resource := range kmsResources {
		resourceNameList[i] = resource.ResourceARN
	}

	return resourceNameList, nil
}

func (d *RDSDatabase) getEnabledEncryptionKeys(resourceNameList []*string) ([]*kms.KeyMetadata, error) {
	var keys []*kms.KeyMetadata

	for _, name := range resourceNameList {
		keyMetadata, err := d.client.kmsGetSymmetricKey(*name)
		if err != nil {
			return nil, err
		}
		if keyMetadata != nil && *keyMetadata.KeyState == kms.KeyStateEnabled {
			keys = append(keys, keyMetadata)
		}
	}

	return keys, nil
}
