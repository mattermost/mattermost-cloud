package main

import (
	"context"
	"fmt"

	"github.com/mattermost/mattermost-cloud/internal/tools/awsv2"
	"github.com/sirupsen/logrus"
)

func main() {
	logger := logrus.New()
	config := awsv2.NewConfig(logger)
	client, err := awsv2.NewClientWithConfig(config, logger)
	if err != nil {
		logger.Fatal(err)
	}

	ctx := context.TODO()

	res, err := client.GetCertificateByTag(
		ctx,
		awsv2.DefaultInstallPrivateCertificatesTagKey,
		awsv2.DefaultInstallPrivateCertificatesTagValue,
	)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Println(*res.ARN)
}
