package aws

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-cloud/model"
)

// RDSDatabaseMigration is a migrated database backed by AWS RDS.
type RDSDatabaseMigration struct {
	awsClient           *Client
	installation        *model.Installation
	clusterInstallation *model.ClusterInstallation
}

// NewRDSDatabaseMigration returns a new RDSDatabase interface.
func NewRDSDatabaseMigration(installation *model.Installation, clusterInstallation *model.ClusterInstallation, awsClient *Client) *RDSDatabaseMigration {
	return &RDSDatabaseMigration{
		awsClient:           awsClient,
		installation:        installation,
		clusterInstallation: clusterInstallation,
	}
}

func (d *RDSDatabaseMigration) authorizeRDSAccessSecurityGroup(masterVPCID, slaveVPCID string) error {
	secGroupsSlave, err := d.awsClient.GetSecurityGroupsWithFilters([]*ec2.Filter{
		{
			Name:   aws.String("vpc-id"),
			Values: []*string{aws.String(slaveVPCID)},
		},
		{
			Name:   aws.String(DefaultDBSecurityGroupTagKey),
			Values: []*string{aws.String(DefaultDBSecurityGroupTagValue)},
		},
	})
	if err != nil {
		return err
	}

	secGroupsMaster, err := d.awsClient.GetSecurityGroupsWithFilters([]*ec2.Filter{
		{
			Name:   aws.String("vpc-id"),
			Values: []*string{aws.String(masterVPCID)},
		},
		{
			Name:   aws.String(DefaultDBSecurityGroupTagKey),
			Values: []*string{aws.String(DefaultDBSecurityGroupTagValue)},
		},
	})
	if err != nil {
		return err
	}

	for _, sgMaster := range secGroupsMaster {
		for _, sgSlave := range secGroupsSlave {
			if strings.Contains(*sgSlave.GroupName, "-db-sg") {
				_, err := d.awsClient.Service().ec2.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
					GroupId: sgMaster.GroupId,
					IpPermissions: []*ec2.IpPermission{
						{
							FromPort:   aws.Int64(3306),
							IpProtocol: aws.String("tcp"),
							ToPort:     aws.Int64(3306),
							UserIdGroupPairs: []*ec2.UserIdGroupPair{
								{
									Description: aws.String("DB migration - access from other RDS instance"),
									GroupId:     sgSlave.GroupId,
								},
							},
						},
					},
				})
				if err != nil {
					return err
				}
				return nil
			}
		}
	}

	return errors.Errorf("could not authorize access from RDS instance %s to RDS instance %s", slaveVPCID, masterVPCID)
}

// Setup sets access from one RDS database to another and enable instances for replication.
func (d *RDSDatabaseMigration) Setup(logger log.FieldLogger) (string, error) {
	err := d.authorizeRDSAccessSecurityGroup(CloudID(d.installation.ID), CloudID(d.clusterInstallation.InstallationID))
	if err != nil {
		return model.DatabaseMigrationStatusError, err
	}

	return model.DatabaseMigrationStatusSetupComplete, nil
}

// Replicate starts the process for replicating an master RDS database. This method must return an resplication status or an error.
func (d *RDSDatabaseMigration) Replicate(logger log.FieldLogger) (string, error) {
	return model.DatabaseMigrationStatusReplicationComplete, nil
}
