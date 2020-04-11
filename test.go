package main

import (
	"flag"
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/kms"
	toolsAWS "github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
)

func main() {
	op := flag.Int("op", 0, "an operation")
	awsID := flag.String("aws-id", "cloud-dkjhx453u7ybir6hdzwor1b5or", "an operation")
	vpcID := flag.String("vpc-id", "vpc-0765bd49d8e3f418e", "an operation")

	flag.Parse()

	logger := log.New()
	logger.SetLevel(logrus.DebugLevel)

	client := toolsAWS.NewAWSClientWithConfig(&aws.Config{
		Region:     aws.String(toolsAWS.DefaultAWSRegion),
		MaxRetries: aws.Int(toolsAWS.DefaultAWSClientRetries),
	}, logger)

	switch *op {
	case 0:
		rdsSecret, err := client.SecretsManagerEnsureRDSSecretCreated(*awsID, logger)
		if err != nil {
			panic(err)
		}
		encryptionKey, err := client.KmsCreateSymmetricKey(*awsID, "Key used for encrypting RDS database")
		if err != nil {
			panic(errors.Wrapf(err, "unable to create RDS encryption key for installation %s", *awsID))
		}
		err = client.KmsCreateAlias(*encryptionKey.KeyId, toolsAWS.KMSAliasNameRDS(*awsID))
		if err != nil && !toolsAWS.IsErrorCode(err, kms.ErrCodeAlreadyExistsException) {
			deletionKeyErr := client.KmsScheduleKeyDeletion(*encryptionKey.KeyId, toolsAWS.KMSMinTimeEncryptionKeyDeletion)
			if deletionKeyErr != nil {
				logger.WithError(deletionKeyErr).Errorf("Failed to schedule encryption key %s for deletition", *encryptionKey.KeyId)
			}
			panic(errors.Wrapf(err, "unable to create a RDS encryption key alias name for installation %s", *awsID))
		}

		err = client.RdsEnsureDBClusterCreated(*awsID, *vpcID, rdsSecret.MasterUsername, rdsSecret.MasterPassword, *encryptionKey.KeyId, logger)
		if err != nil {
			panic(err)
		}

		err = client.RdsEnsureDBClusterInstanceCreated(*awsID, fmt.Sprintf("%s-master", *awsID), logger)
		if err != nil {
			panic(err)
		}
	}
}
