package aws

import (
	"fmt"
	"sort"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-cloud/model"
)

const (
	errMessageTemplate      = "unabled to restore RDS master id %s to RDS replica id %s"
	errDuplicatedPermission = "InvalidPermission.Duplicate"
	errNotFoundPermission   = "InvalidPermission.NotFound"
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
	masterClusterID := CloudID(d.masterInstallationID)
	replicaClusterID := RDSMigrationClusterID(d.masterInstallationID)

	logger = logger.WithFields(logrus.Fields{
		"master-db-cluster":  masterClusterID,
		"replica-db-cluster": replicaClusterID,
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
		err = newErrorWrap(&masterClusterID, &replicaClusterID, err)
		logger.WithError(err).Error("Restoring RDS database migration")
		return "", err
	}
	if len(vpcs) != 1 {
		err := newErrorWrap(&masterClusterID, &replicaClusterID, errors.Errorf("replica cluster expected exactly 1 VPC, but got %d", len(vpcs)))
		logger.WithError(err).Error("Restoring RDS database migration")
		return "", err
	}

	snapshot, err := d.getMostRecentSnapshot(&masterClusterID)
	if err != nil {
		err = newErrorWrap(&masterClusterID, &replicaClusterID, err)
		logger.WithError(err).Error("Restoring RDS database migration")
		return "", err
	}

	switch *snapshot.Status {
	case RDSStatusAvailable:
		logger.Info("Restoring RDS database migration: Proceeding to slave DB cluster creation")
	case RDSStatusCreating:
		return model.DatabaseMigrationStatusRestoreIP, nil
	case RDSStatusModifying:
		err := newErrorWrap(&masterClusterID, &replicaClusterID, errors.Errorf("snapshot id %s should not be modified during migration", *snapshot.DBClusterSnapshotIdentifier))
		logger.WithError(err).Error("Restoring RDS database migration")
		return "", err
	case RDSStatusDeleting:
		err := newErrorWrap(&masterClusterID, &replicaClusterID, errors.Errorf("snapshot id %s should not be deleted during migration", *snapshot.DBClusterSnapshotIdentifier))
		logger.WithError(err).Error("Restoring RDS database migration")
		return "", err
	default:
		err := newErrorWrap(&masterClusterID, &replicaClusterID, errors.Errorf("snapshot id %s has an unknown status %s", *snapshot.DBClusterSnapshotIdentifier, *snapshot.Status))
		logger.WithError(err).Error("Restoring RDS database migration")
		return "", err
	}

	dbClusterOutput, err := d.awsClient.Service().rds.DescribeDBClusters(&rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(replicaClusterID),
	})
	if err != nil {
		switch {
		case IsErrorCode(err, rds.ErrCodeDBClusterNotFoundFault):
			err = d.awsClient.rdsEnsureRestoreDBClusterFromSnapshot(*vpcs[0].VpcId, replicaClusterID, *snapshot.DBClusterSnapshotIdentifier, logger)
			if err != nil {
				err = newErrorWrap(&masterClusterID, &replicaClusterID, err)
				logger.WithError(err).Error("Restoring RDS database migration")
				return "", err
			}
		default:
			err := newErrorWrap(&masterClusterID, &replicaClusterID, err)
			logger.WithError(err).Error("Restoring RDS database migration")
			return "", err
		}
	} else {
		switch *dbClusterOutput.DBClusters[0].Status {
		case RDSStatusAvailable:
			logger.Info("Restoring RDS database migration: Proceeding to slave DB instance creation request")
		case RDSStatusCreating:
			logger.Info("Restoring RDS database migration: Proceeding to slave DB instance creation request")
		case RDSStatusDeleting:
			logger.Debug("Restoring RDS database migration: Waiting until slave db cluster deletion is completed")
			return model.DatabaseMigrationStatusRestoreIP, nil
		case RDSStatusModifying:
			err := newErrorWrap(&masterClusterID, &replicaClusterID, errors.New("slave db cluster should not be modified during migration"))
			logger.WithError(err).Error("Restoring RDS database migration")
			return "", err
		default:
			err := newErrorWrap(&masterClusterID, &replicaClusterID, errors.Errorf("slave db cluster has an unknown status %s", *snapshot.Status))
			logger.WithError(err).Error("Restoring RDS database migration")
			return "", err
		}
	}

	replicaInstanceID := RDSMigrationMasterInstanceID(d.masterInstallationID)

	dbInstanceOutput, err := d.awsClient.Service().rds.DescribeDBInstances(&rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(replicaInstanceID),
	})
	if err != nil {
		switch {
		case IsErrorCode(err, rds.ErrCodeDBInstanceNotFoundFault):
			err = d.awsClient.rdsEnsureDBClusterInstanceCreated(replicaClusterID, replicaInstanceID, logger)
			if err != nil {
				err = newErrorWrap(&masterClusterID, &replicaClusterID, err)
				logger.WithError(err).Error("Restoring RDS database migration")
				return "", err
			}
		default:
			err = newErrorWrap(&masterClusterID, &replicaClusterID, err)
			logger.WithError(err).Error("Restoring RDS database migration")
			return "", err
		}
	} else {
		switch *dbInstanceOutput.DBInstances[0].DBInstanceStatus {
		case RDSStatusDeleting:
			logger.Debug("Restoring RDS database migration: Waiting until slave db instance deletion is completed")
			return model.DatabaseMigrationStatusRestoreIP, nil
		case RDSStatusAvailable:
			err := newErrorWrap(&masterClusterID, &replicaClusterID, errors.Errorf("slave DB instance id %s already exists", replicaInstanceID))
			logger.WithError(err).Error("Restoring RDS database migration")
			return "", err
		case RDSStatusCreating:
			err := newErrorWrap(&masterClusterID, &replicaClusterID, errors.Errorf("slave DB instance id %s already being created", replicaInstanceID))
			logger.WithError(err).Error("Restoring RDS database migration")
			return "", err
		case RDSStatusModifying:
			err := newErrorWrap(&masterClusterID, &replicaClusterID, errors.Errorf("slave DB instance id %s already exists and is being modified", replicaInstanceID))
			logger.WithError(err).Error("Restoring RDS database migration")
			return "", err
		default:
			err = newErrorWrap(&masterClusterID, &replicaClusterID, errors.Errorf("slave DB instance id %s has an unknown status %s", replicaInstanceID, *dbClusterOutput.DBClusters[0].Status))
			logger.WithError(err).Error("Restoring RDS database migration")
			return "", err
		}
	}

	logger.Infof("AWS RDS DB cluster is being restored")
	return model.DatabaseMigrationStatusRestoreComplete, nil
}

// Setup sets access from one RDS database to another and sets any configuration needed for replication.
func (d *RDSDatabaseMigration) Setup(logger log.FieldLogger) (string, error) {
	masterInstanceID := RDSMasterInstanceID(d.masterInstallationID)
	replicaInstanceID := RDSMigrationMasterInstanceID(d.masterInstallationID)

	logger = logger.WithFields(logrus.Fields{
		"master-db-instance":  masterInstanceID,
		"replica-db-instance": replicaInstanceID,
	})

	masterInstanceSG, err := d.describeDBInstanceSecurityGroup(masterInstanceID)
	if err != nil {
		err = newErrorWrap(&masterInstanceID, &replicaInstanceID, err)
		logger.WithError(err).Error("Setting up RDS database migration")
		return "", err
	}

	replicaInstanceSG, err := d.describeDBInstanceSecurityGroup(replicaInstanceID)
	if err != nil {
		err = newErrorWrap(&masterInstanceID, &replicaInstanceID, err)
		logger.WithError(err).Error("Setting up RDS database migration")
		return "", err
	}

	describeVpcsOut, err := d.awsClient.Service().ec2.DescribeVpcs(&ec2.DescribeVpcsInput{
		VpcIds: []*string{replicaInstanceSG.VpcId},
	})
	if err != nil {
		err = newErrorWrap(&masterInstanceID, &replicaInstanceID, err)
		logger.WithError(err).Error("Setting up RDS database migration")
		return "", err
	}
	if len(describeVpcsOut.Vpcs) != 1 {
		err = newErrorWrap(&masterInstanceID, &replicaInstanceID, errors.Errorf("expected DB replica to have exactly 1 VPC, but it got %d", len(describeVpcsOut.Vpcs)))
		logger.WithError(err).Error("Setting up RDS database migration")
		return "", err
	}

	_, err = d.awsClient.Service().ec2.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: masterInstanceSG.GroupId,
		IpPermissions: []*ec2.IpPermission{
			{
				IpRanges: []*ec2.IpRange{
					{
						CidrIp:      describeVpcsOut.Vpcs[0].CidrBlock,
						Description: aws.String(fmt.Sprintf("Ingress Traffic from replica RDS instance %s", replicaInstanceID)),
					},
				},
				FromPort:   aws.Int64(3306),
				IpProtocol: aws.String("tcp"),
				ToPort:     aws.Int64(3306),
			},
		},
	})
	// Why "InvalidPermission.Duplicate" is hardcoded? https://github.com/aws/aws-sdk-go/issues/3235
	if err != nil && !IsErrorCode(err, errDuplicatedPermission) {
		err = newErrorWrap(&masterInstanceID, &replicaInstanceID, err)
		logger.WithError(err).Error("Setting up RDS database migration")
		return "", err
	}

	logger.Info("Database migration setup is completed")
	return model.DatabaseMigrationStatusSetupComplete, nil
}

// Teardown removes access from one RDS database to another and rollback any previous database configuration.
func (d *RDSDatabaseMigration) Teardown(logger log.FieldLogger) (string, error) {
	masterInstanceID := RDSMasterInstanceID(d.masterInstallationID)
	replicaInstanceID := RDSMigrationMasterInstanceID(d.masterInstallationID)

	logger = logger.WithFields(logrus.Fields{
		"master-db-instance":  masterInstanceID,
		"replica-db-instance": replicaInstanceID,
	})

	masterInstanceSG, err := d.describeDBInstanceSecurityGroup(masterInstanceID)
	if err != nil {
		err = newErrorWrap(&masterInstanceID, &replicaInstanceID, err)
		logger.WithError(err).Error("Tearing down RDS database migration")
		return "", err
	}

	replicaInstanceSG, err := d.describeDBInstanceSecurityGroup(replicaInstanceID)
	if err != nil {
		err = newErrorWrap(&masterInstanceID, &replicaInstanceID, err)
		logger.WithError(err).Error("Tearing down RDS database migration")
		return "", err
	}

	describeVpcsOut, err := d.awsClient.Service().ec2.DescribeVpcs(&ec2.DescribeVpcsInput{
		VpcIds: []*string{replicaInstanceSG.VpcId},
	})
	if err != nil {
		err = newErrorWrap(&masterInstanceID, &replicaInstanceID, err)
		logger.WithError(err).Error("Tearing down RDS database migration")
		return "", err
	}
	if len(describeVpcsOut.Vpcs) != 1 {
		err = newErrorWrap(&masterInstanceID, &replicaInstanceID, errors.Errorf("expected DB replica to have exactly 1 VPC, but it got %d", len(describeVpcsOut.Vpcs)))
		logger.WithError(err).Error("Tearing down RDS database migration")
		return "", err
	}

	_, err = d.awsClient.Service().ec2.RevokeSecurityGroupIngress(&ec2.RevokeSecurityGroupIngressInput{
		GroupId: masterInstanceSG.GroupId,
		IpPermissions: []*ec2.IpPermission{
			{
				FromPort:   aws.Int64(3306),
				IpProtocol: aws.String("tcp"),
				ToPort:     aws.Int64(3306),
				IpRanges: []*ec2.IpRange{
					{
						CidrIp: describeVpcsOut.Vpcs[0].CidrBlock,
					},
				},
			},
		},
	})
	// Why "InvalidPermission.NotFound" is hardcoded? https://github.com/aws/aws-sdk-go/issues/3235
	if err != nil && !IsErrorCode(err, errNotFoundPermission) {
		err = newErrorWrap(&masterInstanceID, &replicaInstanceID, err)
		logger.WithError(err).Error("Tearing down RDS database migration")
		return "", err
	}

	logger.Info("Database migration teardown is completed")
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

func isRDSInstanceSecurityGroup(securityGroup *ec2.SecurityGroup) bool {
	for _, tag := range securityGroup.Tags {
		if *tag.Key == trimTagPrefix(DefaultDBSecurityGroupTagKey) && *tag.Value == DefaultDBSecurityGroupTagValue {
			return true
		}
	}
	return false
}

func (d *RDSDatabaseMigration) getMostRecentSnapshot(masterCloudID *string) (*rds.DBClusterSnapshot, error) {
	dbClusterSnapshotsOut, err := d.awsClient.Service().rds.DescribeDBClusterSnapshots(&rds.DescribeDBClusterSnapshotsInput{
		SnapshotType: aws.String(RDSDefaultSnapshotType),
	})
	if err != nil {
		return nil, err
	}

	expectedTagValue := RDSSnapshotTagValue(*masterCloudID)

	var snapshots []*rds.DBClusterSnapshot
	for _, snapshot := range dbClusterSnapshotsOut.DBClusterSnapshots {
		tags, err := d.awsClient.Service().rds.ListTagsForResource(&rds.ListTagsForResourceInput{
			ResourceName: snapshot.DBClusterSnapshotArn,
		})
		if err != nil {
			return nil, err
		}
		for _, tag := range tags.TagList {
			if tag.Key != nil && tag.Value != nil && *tag.Key == DefaultClusterInstallationSnapshotTagKey &&
				*tag.Value == expectedTagValue {
				snapshots = append(snapshots, snapshot)
			}
		}
	}
	if len(snapshots) < 1 {
		return nil, errors.Errorf("DB cluster %s has no snapshots", *masterCloudID)
	}
	sort.SliceStable(snapshots, func(i, j int) bool {
		return snapshots[i].SnapshotCreateTime.After(*snapshots[j].SnapshotCreateTime)
	})

	return snapshots[0], nil
}

func newError(masterClusterID, replicaClusterID *string, message string) error {
	return errors.Wrap(errors.Errorf(errMessageTemplate, *masterClusterID, *replicaClusterID), message)
}

func newErrorWrap(masterClusterID, replicaClusterID *string, err error) error {
	return errors.Wrap(err, fmt.Sprintf(errMessageTemplate, *masterClusterID, *replicaClusterID))
}

func newErrorWrapWithMessage(masterClusterID, replicaClusterID *string, err error, message string) error {
	return errors.Wrap(newErrorWrap(masterClusterID, replicaClusterID, err), message)
}
