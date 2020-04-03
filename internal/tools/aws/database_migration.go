package aws

import (
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-cloud/model"
)

// RDSDatabaseMigration is a migrated database backed by AWS RDS.
type RDSDatabaseMigration struct {
	awsClient            *Client
	masterInstallationID string
	slaveClusterID       string
}

// NewRDSDatabaseMigration returns a new RDSDatabaseMigration.
func NewRDSDatabaseMigration(masterInstallationID, slaveClusterID string, awsClient *Client) *RDSDatabaseMigration {
	return &RDSDatabaseMigration{
		awsClient:            awsClient,
		masterInstallationID: masterInstallationID,
		slaveClusterID:       slaveClusterID,
	}
}

// Restore restores database from the most recent snapshot.
func (d *RDSDatabaseMigration) Restore(logger log.FieldLogger) (string, error) {
	masterCloudID := CloudID(d.masterInstallationID)

	logger = logger.WithFields(logrus.Fields{
		"master-db-cluster": masterCloudID,
	})

	vpcs, err := d.awsClient.GetVpcsWithFilters([]*ec2.Filter{
		{
			Name:   aws.String(VpcClusterIDTagKey),
			Values: []*string{aws.String(d.slaveClusterID)},
		},
		{
			Name:   aws.String(VpcAvailableTagKey),
			Values: []*string{aws.String(VpcAvailableTagValueFalse)},
		},
	})
	if err != nil {
		return "", errors.Wrap(err, "unabled to restore RDS database")
	}
	if len(vpcs) != 1 {
		return "", errors.Errorf("unabled to restore RDS database: expected 1 VPC in cluster id %s, but got %d", d.slaveClusterID, len(vpcs))
	}

	dbClusterSnapshotsOut, err := d.awsClient.Service().rds.DescribeDBClusterSnapshots(&rds.DescribeDBClusterSnapshotsInput{
		SnapshotType: aws.String(RDSDefaultSnapshotType),
	})
	if err != nil {
		return "", errors.Wrap(err, "unabled to restore RDS database")
	}

	expectedTagValue := RDSSnapshotTagValue(masterCloudID)

	var snapshots []*rds.DBClusterSnapshot
	for _, snapshot := range dbClusterSnapshotsOut.DBClusterSnapshots {
		tags, err := d.awsClient.Service().rds.ListTagsForResource(&rds.ListTagsForResourceInput{
			ResourceName: snapshot.DBClusterSnapshotArn,
		})
		if err != nil {
			return "", errors.Wrap(err, "unabled to restore RDS database")
		}
		for _, tag := range tags.TagList {
			if tag.Key != nil && tag.Value != nil && *tag.Key == DefaultClusterInstallationSnapshotTagKey &&
				*tag.Value == expectedTagValue {
				snapshots = append(snapshots, snapshot)
			}
		}
	}
	if len(snapshots) < 1 {
		return "", errors.Errorf("unabled to restore RDS database: DB cluster %s has no snapshots", masterCloudID)
	}
	sort.SliceStable(snapshots, func(i, j int) bool {
		return snapshots[i].SnapshotCreateTime.After(*snapshots[j].SnapshotCreateTime)
	})

	switch *snapshots[0].Status {
	case RDSStatusAvailable:
		logger.Info("Snapshot of master database is complete. Proceeding to slave DB cluster creation")
	case RDSStatusCreating:
		return model.DatabaseMigrationStatusRestoreIP, nil
	case RDSStatusModifying:
		return "", errors.Errorf("unabled to restore RDS database: snapshot id %s is been modified", *snapshots[0].DBClusterSnapshotIdentifier)
	case RDSStatusDeleting:
		return "", errors.Errorf("unabled to restore RDS database: snapshot id %s is been deleted", *snapshots[0].DBClusterSnapshotIdentifier)
	default:
		return "", errors.Errorf("unabled to restore RDS database: snapshot id %s has an unknown status %s", *snapshots[0].DBClusterSnapshotIdentifier, *snapshots[0].Status)
	}

	migrationClusterID := RDSMigrationClusterID(d.masterInstallationID)

	dbClusterOutput, err := d.awsClient.Service().rds.DescribeDBClusters(&rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(migrationClusterID),
	})
	if err != nil {
		switch {
		case IsErrorCode(err, rds.ErrCodeDBClusterNotFoundFault):
			err = d.awsClient.rdsEnsureRestoreDBClusterFromSnapshot(*vpcs[0].VpcId, migrationClusterID, *snapshots[0].DBClusterSnapshotIdentifier, logger)
			if err != nil {
				return "", errors.Wrapf(err, "unabled to restore RDS database %s", masterCloudID)
			}
		default:
			return "", errors.Wrapf(err, "unabled to restore database from DB cluster %s", masterCloudID)
		}
	} else {
		switch *dbClusterOutput.DBClusters[0].Status {
		case RDSStatusAvailable:
			logger.Debug("Slave DB cluster is available: proceeding to slave DB instance creation request")
		case RDSStatusCreating:
			logger.Debug("Slave DB cluster creation in progress: proceeding to slave DB instance creation request")
		case RDSStatusDeleting:
			logger.Debug("Slave DB cluster deletion in progress: waiting until it is done")
			return model.DatabaseMigrationStatusRestoreIP, nil
		case RDSStatusModifying:
			return "", errors.Errorf("unabled to restore RDS database: RDS is modifying slave DB cluster id %s", migrationClusterID)
		default:
			return "", errors.Errorf("unabled to restore RDS database: DB cluster id %s has an unknown status %s", migrationClusterID, *snapshots[0].Status)
		}
	}

	migrationInstanceID := RDSMigrationMasterInstanceID(d.masterInstallationID)

	dbInstanceOutput, err := d.awsClient.Service().rds.DescribeDBInstances(&rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(migrationInstanceID),
	})
	if err != nil {
		switch {
		case IsErrorCode(err, rds.ErrCodeDBInstanceNotFoundFault):
			err = d.awsClient.rdsEnsureDBClusterInstanceCreated(migrationClusterID, migrationInstanceID, logger)
			if err != nil {
				return "", errors.Wrapf(err, "unabled to restore database from DB cluster %s", masterCloudID)
			}
		default:
			return "", errors.Wrapf(err, "unabled to restore database from DB cluster %s", masterCloudID)
		}
	} else {
		switch *dbInstanceOutput.DBInstances[0].DBInstanceStatus {
		case RDSStatusDeleting:
			logger.Debug("Slave DB cluster deletion in progress: waiting until it is done")
			return model.DatabaseMigrationStatusRestoreIP, nil
		case RDSStatusAvailable:
			return "", errors.Errorf("unabled to restore RDS database: slave DB instance id %s already exists", migrationInstanceID)
		case RDSStatusCreating:
			return "", errors.Errorf("unabled to restore RDS database: slave DB instance id %s already exists", migrationInstanceID)
		case RDSStatusModifying:
			return "", errors.Errorf("unabled to restore RDS database: slave DB instance id %s already exists", migrationInstanceID)
		default:
			return "", errors.Errorf("unabled to restore RDS database: DB instance id %s has an unknown status %s", migrationInstanceID, *dbClusterOutput.DBClusters[0].Status)
		}
	}

	logger.WithFields(logrus.Fields{
		"master-db-cluster": masterCloudID,
	}).Infof("AWS RDS DB cluster is being restored from %s", masterCloudID)

	return model.DatabaseMigrationStatusRestoreComplete, nil
}

// Setup sets access from one RDS database to another and sets any configuration needed for replication.
func (d *RDSDatabaseMigration) Setup(logger log.FieldLogger) (string, error) {
	masterInstanceSG, err := d.describeDBInstanceSecurityGroup(RDSMasterInstanceID(d.masterInstallationID))
	if err != nil {
		return "", d.toSetupError(err)
	}

	slaveInstanceSG, err := d.describeDBInstanceSecurityGroup(RDSMigrationMasterInstanceID(d.masterInstallationID))
	if err != nil {
		return "", d.toSetupError(err)
	}

	_, err = d.awsClient.Service().ec2.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: masterInstanceSG.GroupId,
		IpPermissions: []*ec2.IpPermission{
			{
				FromPort:   aws.Int64(3306),
				IpProtocol: aws.String("tcp"),
				ToPort:     aws.Int64(3306),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						Description: aws.String("Ingress Traffic from other RDS instance"),
						GroupId:     slaveInstanceSG.GroupId,
					},
				},
			},
		},
	})
	// Why "InvalidPermission.Duplicate" is hardcoded? https://github.com/aws/aws-sdk-go/issues/3235
	if err != nil && !IsErrorCode(err, "InvalidPermission.Duplicate") {
		return "", d.toSetupError(err)
	}

	logger.WithFields(logrus.Fields{
		"master-installation-id": d.masterInstallationID,
	}).Info("Database migration setup completed")

	return model.DatabaseMigrationStatusSetupComplete, nil
}

// Teardown removes access from one RDS database to another and rollback any previous database configuration.
func (d *RDSDatabaseMigration) Teardown(logger log.FieldLogger) (string, error) {
	masterInstanceSG, err := d.describeDBInstanceSecurityGroup(RDSMasterInstanceID(d.masterInstallationID))
	if err != nil {
		return "", d.toTeardownError(err)
	}

	slaveInstanceSG, err := d.describeDBInstanceSecurityGroup(RDSMigrationMasterInstanceID(d.masterInstallationID))
	if err != nil {
		return "", d.toTeardownError(err)
	}

	_, err = d.awsClient.Service().ec2.RevokeSecurityGroupIngress(&ec2.RevokeSecurityGroupIngressInput{
		GroupId: masterInstanceSG.GroupId,
		IpPermissions: []*ec2.IpPermission{
			{
				FromPort:   aws.Int64(3306),
				IpProtocol: aws.String("tcp"),
				ToPort:     aws.Int64(3306),
				UserIdGroupPairs: []*ec2.UserIdGroupPair{
					{
						GroupId: slaveInstanceSG.GroupId,
					},
				},
			},
		},
	})
	// Why "InvalidPermission.NotFound" is hardcoded? https://github.com/aws/aws-sdk-go/issues/3235
	if err != nil && !IsErrorCode(err, "InvalidPermission.NotFound") {
		return "", d.toTeardownError(err)
	}

	logger.WithFields(logrus.Fields{
		"master-installation-id": d.masterInstallationID,
	}).Info("Database migration teardown completed")

	return model.DatabaseMigrationStatusTeardownComplete, nil
}

// Replicate starts the process for replicating an master RDS database. This method must return an
// resplication status or an error.
func (d *RDSDatabaseMigration) Replicate(logger log.FieldLogger) (string, error) {
	return "", errors.New("not implemented")
}

func (d *RDSDatabaseMigration) describeDBInstanceSecurityGroup(instanceID string) (*ec2.SecurityGroup, error) {
	output, err := d.awsClient.Service().rds.DescribeDBInstances(&rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceID),
	})
	if err != nil {
		return nil, err
	}

	for _, instance := range output.DBInstances {
		for _, vpcSG := range instance.VpcSecurityGroups {
			sgOutput, err := d.awsClient.Service().ec2.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
				GroupIds: []*string{vpcSG.VpcSecurityGroupId},
			})
			if err != nil {
				return nil, err
			}
			if len(sgOutput.SecurityGroups) == 1 && isRDSInstanceSecurityGroup(sgOutput.SecurityGroups[0]) {
				return sgOutput.SecurityGroups[0], nil
			}
		}
	}

	return nil, errors.Errorf("security group for RDS DB instance %s not found", instanceID)
}

func (d *RDSDatabaseMigration) toSetupError(err error) error {
	return errors.Wrapf(err, "unable to setup database migration for master installation id: %s and to slave installation id: %s",
		d.masterInstallationID, d.masterInstallationID)
}

func (d *RDSDatabaseMigration) toTeardownError(err error) error {
	return errors.Wrapf(err, "unable to setup database migration for master installation id: %s and to slave installation id: %s",
		d.masterInstallationID, d.masterInstallationID)
}

func isRDSInstanceSecurityGroup(securityGroup *ec2.SecurityGroup) bool {
	for _, tag := range securityGroup.Tags {
		if *tag.Key == trimTagPrefix(DefaultDBSecurityGroupTagKey) && *tag.Value == DefaultDBSecurityGroupTagValue {
			return true
		}
	}
	return false
}
