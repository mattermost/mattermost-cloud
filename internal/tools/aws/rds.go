// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/rds"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (a *Client) rdsGetDBSecurityGroupIDs(vpcID, tagValue string, logger log.FieldLogger) ([]string, error) {
	result, err := a.Service().ec2.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		Filters: []*ec2.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []*string{aws.String(vpcID)},
			},
			{
				Name:   aws.String(DefaultDBSecurityGroupTagKey),
				Values: []*string{aws.String(tagValue)},
			},
		},
	})
	if err != nil {
		return []string{}, err
	}

	var dbSecurityGroups []string
	for _, sg := range result.SecurityGroups {
		dbSecurityGroups = append(dbSecurityGroups, *sg.GroupId)
	}

	if len(dbSecurityGroups) == 0 {
		return []string{}, fmt.Errorf("unable to find security groups tagged for Mattermost DB usage: %s=%s", DefaultDBSecurityGroupTagKey, DefaultDBSecurityGroupTagMySQLValue)
	}

	logger.WithField("security-group-ids", dbSecurityGroups).Debugf("Found %d DB tagged security groups", len(dbSecurityGroups))

	return dbSecurityGroups, nil
}

func (a *Client) rdsGetDBSubnetGroupName(vpcID string, logger log.FieldLogger) (string, error) {
	// TODO:
	// The subnet group describe functionality does not currently support
	// filters. Instead, we look up all the subnet groups and match based on
	// name. The name format is based on our terraform creation logic.
	// Example Name: mattermost-provisioner-db-vpc-VPC_ID_HERE
	//
	// We should periodically check if filters become supported and move to that
	// when they do.
	result, err := a.Service().rds.DescribeDBSubnetGroups(nil)
	if err != nil {
		return "", err
	}

	for _, subnetGroup := range result.DBSubnetGroups {
		// AWS names are unique, so there will only be one that correctly matches.
		if *subnetGroup.DBSubnetGroupName == DBSubnetGroupName(vpcID) {
			name := *subnetGroup.DBSubnetGroupName
			logger.WithField("db-subnet-group-name", name).Debugf("Found DB subnet group")

			return name, nil
		}
	}

	return "", fmt.Errorf("unable to find subnet group tagged for Mattermost DB usage: %s=%s", DefaultDBSubnetGroupTagKey, DefaultDBSubnetGroupTagValue)
}

func (a *Client) rdsEnsureDBClusterCreated(awsID, vpcID, username, password, kmsKeyID, databaseType string, logger log.FieldLogger) error {
	var engine, engineVersion, sgTagValue string
	var port int64
	switch databaseType {
	case model.DatabaseEngineTypeMySQL:
		engine = "aurora-mysql"
		engineVersion = DefaultDatabaseMySQLVersion
		port = 3306
		sgTagValue = DefaultDBSecurityGroupTagMySQLValue
	case model.DatabaseEngineTypePostgres:
		engine = "aurora-postgresql"
		engineVersion = DefaultDatabasePostgresVersion
		port = 5432
		sgTagValue = DefaultDBSecurityGroupTagPostgresValue
	default:
		return errors.Errorf("%s is an invalid database engine type", databaseType)
	}

	_, err := a.Service().rds.DescribeDBClusters(&rds.DescribeDBClustersInput{
		DBClusterIdentifier: aws.String(awsID),
	})
	if err == nil {
		logger.WithField("db-cluster-name", awsID).Debug("AWS DB cluster already created")

		return nil
	}

	dbSecurityGroupIDs, err := a.rdsGetDBSecurityGroupIDs(vpcID, sgTagValue, logger)
	if err != nil {
		return err
	}

	dbSubnetGroupName, err := a.rdsGetDBSubnetGroupName(vpcID, logger)
	if err != nil {
		return err
	}

	azs, err := a.getAvailabilityZones()
	if err != nil {
		return err
	}

	// default to at least 2 AZ
	rdsAZs := azs[0:2]
	if len(azs) >= 3 {
		rand.Seed(time.Now().UnixNano())
		rand.Shuffle(len(azs), func(i, j int) { azs[i], azs[j] = azs[j], azs[i] })
		rdsAZs = azs[0:3]
	}

	input := &rds.CreateDBClusterInput{
		AvailabilityZones:     rdsAZs,
		BackupRetentionPeriod: aws.Int64(7),
		DBClusterIdentifier:   aws.String(awsID),
		DatabaseName:          aws.String("mattermost"),
		EngineMode:            aws.String("provisioned"),
		Engine:                aws.String(engine),
		EngineVersion:         aws.String(engineVersion),
		MasterUserPassword:    aws.String(password),
		MasterUsername:        aws.String(username),
		Port:                  aws.Int64(port),
		StorageEncrypted:      aws.Bool(true),
		DBSubnetGroupName:     aws.String(dbSubnetGroupName),
		VpcSecurityGroupIds:   aws.StringSlice(dbSecurityGroupIDs),
		KmsKeyId:              aws.String(kmsKeyID),
	}

	_, err = a.Service().rds.CreateDBCluster(input)
	if err != nil {
		return err
	}

	logger.WithField("db-cluster-name", awsID).Debug("AWS DB cluster created")

	return nil
}

func (a *Client) rdsEnsureDBClusterInstanceCreated(awsID, instanceName, databaseType string, logger log.FieldLogger) error {
	_, err := a.Service().rds.DescribeDBInstances(&rds.DescribeDBInstancesInput{
		DBInstanceIdentifier: aws.String(instanceName),
	})
	if err == nil {
		logger.WithField("db-instance-name", instanceName).Debug("AWS DB instance already created")

		return nil
	}

	// Some settings have to be tailored to the engine type like instance class:
	// https://docs.aws.amazon.com/AmazonRDS/latest/AuroraUserGuide/Concepts.DBInstanceClass.html
	var engine, instanceClass string
	switch databaseType {
	case model.DatabaseEngineTypeMySQL:
		engine = "aurora-mysql"
		instanceClass = "db.t3.small"
	case model.DatabaseEngineTypePostgres:
		engine = "aurora-postgresql"
		instanceClass = "db.r5.large"
	default:
		return errors.Errorf("%s is an invalid database engine type", databaseType)
	}

	_, err = a.Service().rds.CreateDBInstance(&rds.CreateDBInstanceInput{
		DBClusterIdentifier:  aws.String(awsID),
		DBInstanceIdentifier: aws.String(instanceName),
		DBInstanceClass:      aws.String(instanceClass),
		Engine:               aws.String(engine),
		PubliclyAccessible:   aws.Bool(false),
	})
	if err != nil {
		return err
	}

	logger.WithField("db-instance-name", instanceName).Debug("AWS DB instance created")

	return nil
}

func (a *Client) rdsEnsureDBClusterDeleted(awsID string, logger log.FieldLogger) error {
	result, err := a.Service().rds.DescribeDBClusters(&rds.DescribeDBClustersInput{
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
		_, err = a.Service().rds.DeleteDBInstance(&rds.DeleteDBInstanceInput{
			DBInstanceIdentifier: instance.DBInstanceIdentifier,
			SkipFinalSnapshot:    aws.Bool(true),
		})
		if err != nil {
			return errors.Wrap(err, "unable to delete DB cluster instance")
		}
		logger.WithField("db-instance-name", *instance.DBInstanceIdentifier).Debug("DB instance deleted")
	}

	_, err = a.Service().rds.DeleteDBCluster(&rds.DeleteDBClusterInput{
		DBClusterIdentifier: aws.String(awsID),
		SkipFinalSnapshot:   aws.Bool(true),
	})
	if err != nil {
		return errors.Wrap(err, "unable to delete DB cluster")
	}

	logger.WithField("db-cluster-name", awsID).Debug("DBCluster deleted")

	return nil
}
