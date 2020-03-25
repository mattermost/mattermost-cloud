package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-cloud/model"
)

// RDSDatabaseMigration is a migrated database backed by AWS RDS.
type RDSDatabaseMigration struct {
	awsClient            *Client
	masterInstallationID string
	slaveInstallationID  string
}

// NewRDSDatabaseMigration returns a new RDSDatabaseMigration.
func NewRDSDatabaseMigration(masterInstallationID, slaveInstallationID string, awsClient *Client) *RDSDatabaseMigration {
	return &RDSDatabaseMigration{
		awsClient:            awsClient,
		masterInstallationID: masterInstallationID,
		slaveInstallationID:  slaveInstallationID,
	}
}

// Setup sets access from one RDS database to another and sets any configuration needed for replication.
func (d *RDSDatabaseMigration) Setup(logger log.FieldLogger) (string, error) {
	masterInstanceSG, err := d.describeDBInstanceSecurityGroup(RDSMasterInstanceID(d.masterInstallationID))
	if err != nil {
		return "", errors.Wrapf(err, "unable to setup database migration for installation id: %s", d.masterInstallationID)
	}

	slaveInstanceSG, err := d.describeDBInstanceSecurityGroup(RDSMigrationInstanceID(d.slaveInstallationID))
	if err != nil {
		return "", errors.Wrapf(err, "unable to setup database migration for installation id: %s", d.masterInstallationID)
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
	if err != nil && !isSecurityGroupRuleDuplicate(err) {
		return "", errors.Wrapf(err, "unable to setup database migration for installation id: %s", d.masterInstallationID)
	}

	logger.WithField("migration-installation-id", d.masterInstallationID).Info("Database migration setup completed")

	return model.DatabaseMigrationStatusSetupComplete, nil
}

// Teardown removes access from one RDS database to another and rollback any previous database configuration.
func (d *RDSDatabaseMigration) Teardown(logger log.FieldLogger) (string, error) {
	masterInstanceSG, err := d.describeDBInstanceSecurityGroup(RDSMasterInstanceID(d.masterInstallationID))
	if err != nil {
		return "", errors.Wrapf(err, "unable to teardown database migration for installation id: %s", d.masterInstallationID)
	}

	slaveInstanceSG, err := d.describeDBInstanceSecurityGroup(RDSMigrationInstanceID(d.slaveInstallationID))
	if err != nil {
		return "", errors.Wrapf(err, "unable to teardown database migration for installation id: %s", d.masterInstallationID)
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
	if err != nil && !isSecurityGroupRuleNotFound(err) {
		return "", errors.Wrapf(err, "unable to teardown database migration for installation id: %s", d.masterInstallationID)
	}

	logger.WithField("migration-installation-id", d.masterInstallationID).Info("Database migration teardown completed")

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

func isSecurityGroupRuleNotFound(err error) bool {
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/errors-overview.html#CommonErrors
		case "InvalidPermission.NotFound":
			return true
		}
	}
	return false
}

func isSecurityGroupRuleDuplicate(err error) bool {
	if aerr, ok := err.(awserr.Error); ok {
		switch aerr.Code() {
		// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/errors-overview.html#CommonErrors
		case "InvalidPermission.Duplicate":
			return true
		}
	}
	return false
}

func isRDSInstanceSecurityGroup(securityGroup *ec2.SecurityGroup) bool {
	for _, tag := range securityGroup.Tags {
		if *tag.Key == trimTagPrefix(DefaultDBSecurityGroupTagKey) && *tag.Value == DefaultDBSecurityGroupTagValue {
			return true
		}
	}
	return false
}
