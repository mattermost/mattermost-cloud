package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (a *Client) rdsGetDBSecurityGroupIDs(vpcID string, logger log.FieldLogger) ([]string, error) {
	svc := ec2.New(session.New(), &aws.Config{
		Region: aws.String(DefaultAWSRegion),
	})

	input := &ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(vpcID)},
			},
			{
				Name:   aws.String("tag:DatabaseType"),
				Values: []*string{aws.String("MYSQL/Aurora")},
			},
		},
	}

	result, err := svc.DescribeSecurityGroups(input)
	if err != nil {
		return []string{}, err
	}

	var dbSecurityGroups []string
	for _, sg := range result.SecurityGroups {
		dbSecurityGroups = append(dbSecurityGroups, *sg.GroupId)
	}

	if len(dbSecurityGroups) == 0 {
		return []string{}, fmt.Errorf("unable to find security groups tagged for Mattermost DB usage: %s=%s", DefaultDBSecurityGroupTagKey, DefaultDBSecurityGroupTagValue)
	}

	logger.WithField("security-group-ids", dbSecurityGroups).Debugf("Found %d DB tagged security groups", len(dbSecurityGroups))

	return dbSecurityGroups, nil
}

func (a *Client) rdsGetDBSubnetGroupName(vpcID string, logger log.FieldLogger) (string, error) {
	svc := rds.New(session.New(), &aws.Config{
		Region: aws.String(DefaultAWSRegion),
	})

	input := &rds.DescribeDBSubnetGroupsInput{
		Filters: []*rds.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(vpcID)},
			},
			{
				Name:   aws.String("tag:DBSubnetGroupType"),
				Values: []*string{aws.String("provisioning")},
			},
		},
	}

	result, err := svc.DescribeDBSubnetGroups(input)
	if err != nil {
		return "", err
	}

	if len(result.DBSubnetGroups) != 1 {
		return "", fmt.Errorf("unable to find security groups tagged for Mattermost DB usage: %s=%s", DefaultDBSecurityGroupTagKey, DefaultDBSecurityGroupTagValue)
	}

	name := *result.DBSubnetGroups[0].DBSubnetGroupName
	logger.WithField("db-subnet-group-name", name).Debugf("Found DB subnet group")

	return name, nil
}

func (a *Client) rdsEnsureDBClusterCreated(awsID, vpcID, username, password string, logger log.FieldLogger) error {
	svc := rds.New(session.New(), &aws.Config{
		Region: aws.String(DefaultAWSRegion),
	})

	_, err := svc.DescribeDBClusters(&rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(awsID),
	})
	if err == nil {
		logger.WithField("db-cluster-name", awsID).Debug("AWS DB cluster already created")

		return nil
	}

	//DB Subnet Group -> DBSubnetGroupType: provisioning
	dbSecurityGroupIDs, err := a.rdsGetDBSecurityGroupIDs(vpcID, logger)
	if err != nil {
		return err
	}

	dbSubnetGroupName, err := a.rdsGetDBSubnetGroupName(vpcID, logger)
	if err != nil {
		return err
	}

	input := &rds.CreateDBClusterInput{
		AvailabilityZones: []*string{
			aws.String("us-east-1a"),
			aws.String("us-east-1b"),
			aws.String("us-east-1c"),
		},
		BackupRetentionPeriod: aws.Int64(7),
		DBClusterIdentifier:   aws.String(awsID),
		DatabaseName:          aws.String("mattermost"),
		EngineMode:            aws.String("provisioned"),
		Engine:                aws.String("aurora-mysql"),
		EngineVersion:         aws.String("5.7"),
		MasterUserPassword:    aws.String(password),
		MasterUsername:        aws.String(username),
		Port:                  aws.Int64(3306),
		StorageEncrypted:      aws.Bool(false),
		DBSubnetGroupName:     aws.String(dbSubnetGroupName),
		VpcSecurityGroupIds:   aws.StringSlice(dbSecurityGroupIDs),
	}

	_, err = svc.CreateDBCluster(input)
	if err != nil {
		return err
	}

	logger.WithField("db-cluster-name", awsID).Debug("AWS DB cluster created")

	return nil
}

func (a *Client) rdsEnsureDBClusterInstanceCreated(awsID, instanceName string, logger log.FieldLogger) error {
	svc := rds.New(session.New(), &aws.Config{
		Region: aws.String(DefaultAWSRegion),
	})

	_, err := svc.DescribeDBInstances(&rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceName),
	})
	if err == nil {
		logger.WithField("db-instance-name", instanceName).Debug("AWS DB instance already created")

		return nil
	}

	_, err = svc.CreateDBInstance(&rds.CreateDBInstanceInput{
		DBClusterIdentifier:  aws.String(awsID),
		DBInstanceIdentifier: aws.String(instanceName),
		DBInstanceClass:      aws.String("db.r5.large"),
		Engine:               aws.String("aurora-mysql"),
		PubliclyAccessible:   aws.Bool(false),
	})
	if err != nil {
		return err
	}

	logger.WithField("db-instance-name", instanceName).Debug("AWS DB instance created")

	return nil
}

func rdsGetDBCluster(awsID string, logger log.FieldLogger) (*rds.DBCluster, error) {
	svc := rds.New(session.New(), &aws.Config{
		Region: aws.String(DefaultAWSRegion),
	})

	result, err := svc.DescribeDBClusters(&rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(awsID),
	})
	if err != nil {
		return nil, err
	}

	if len(result.DBClusters) != 1 {
		return nil, fmt.Errorf("expected 1 DB cluster, but got %d", len(result.DBClusters))
	}

	return result.DBClusters[0], nil
}

func (a *Client) rdsEnsureDBClusterDeleted(awsID string, logger log.FieldLogger) error {
	svc := rds.New(session.New(), &aws.Config{
		Region: aws.String(DefaultAWSRegion),
	})

	result, err := svc.DescribeDBClusters(&rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(awsID),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == rds.ErrCodeDBClusterNotFoundFault {
				logger.WithField("db-cluster-name", awsID).Warn("DBCluster could not be found; assuming already deleted")

				return nil
			}
		}
		return err
	}

	if len(result.DBClusters) != 1 {
		return fmt.Errorf("expected 1 DB cluster, but got %d", len(result.DBClusters))
	}

	for _, instance := range result.DBClusters[0].DBClusterMembers {
		_, err = svc.DeleteDBInstance(&rds.DeleteDBInstanceInput{
			DBInstanceIdentifier: instance.DBInstanceIdentifier,
			SkipFinalSnapshot:    aws.Bool(true),
		})
		if err != nil {
			return errors.Wrap(err, "unable to delete DB cluster instance")
		}
		logger.WithField("db-instance-name", *instance.DBInstanceIdentifier).Debug("DB instance deleted")
	}

	_, err = svc.DeleteDBCluster(&rds.DeleteDBClusterInput{
		DBClusterIdentifier: aws.String(awsID),
		SkipFinalSnapshot:   aws.Bool(true),
	})
	if err != nil {
		return errors.Wrap(err, "unable to delete DB cluster")
	}

	logger.WithField("db-cluster-name", awsID).Debug("DBCluster deleted")

	return nil
}
