package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/rds"
	log "github.com/sirupsen/logrus"
)

func (a *Client) rdsEnsureDBClusterCreated(awsID string, logger log.FieldLogger) error {
	svc := rds.New(session.New())
	input := &rds.CreateDBClusterInput{
		AvailabilityZones: []*string{
			aws.String("us-east-1a"),
		},
		BackupRetentionPeriod:       aws.Int64(1),
		DBClusterIdentifier:         aws.String("mydbcluster"),
		DBClusterParameterGroupName: aws.String("mydbclusterparametergroup"),
		DatabaseName:                aws.String("myauroradb"),
		Engine:                      aws.String("aurora"),
		EngineVersion:               aws.String("5.6.10a"),
		MasterUserPassword:          aws.String("mypassword"),
		MasterUsername:              aws.String("myuser"),
		Port:                        aws.Int64(3306),
		StorageEncrypted:            aws.Bool(true),
	}

	fmt.Println(svc.AddDebugHandlers() + input)

	return nil
}
