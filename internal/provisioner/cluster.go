package provisioner

import (
	"fmt"
	"os"
	"strings"

	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/internal/tools/terraform"
)

// ProviderAWS is the cloud provider AWS.
const ProviderAWS = "aws"

// SizeAlef500 is a cluster sized for 500 users.
const SizeAlef500 = "SizeAlef500"

// CreateCluster creates a cluster using kops and terraform.
func CreateCluster(provider, size string, logger log.FieldLogger) error {
	provider = strings.ToLower(provider)
	if provider != ProviderAWS {
		return fmt.Errorf("unsupported provider %s", provider)
	}

	if size != SizeAlef500 {
		return fmt.Errorf("unsupported size %s", size)
	}

	clusterId := model.NewId()
	s3StateStore := "dev.cloud.mattermost.com"
	dns := fmt.Sprintf("%s-kops.k8s.local", clusterId)

	logger = logger.WithField("cluster", clusterId)

	logger.WithField("dns", dns).Info("creating cluster")

	kops, err := kops.New(s3StateStore, logger)
	if err != nil {
		return err
	}
	defer kops.Close()
	err = kops.CreateCluster(dns, provider, []string{"us-east-1a"})
	if err != nil {
		return err
	}

	// Temporarily relocate the kops output directory to a local folder named `tmp`. This won't
	// be necessary once we persist the output to S3 instead.
	outputDir := "tmp"
	_, err = os.Stat("tmp")
	if err == nil {
		return errors.New("tmp folder already exists: delete existing cluster first")
	} else if err != nil && !os.IsNotExist(err) {
		return errors.Wrap(err, "failed to stat temporary output directory")
	}

	err = os.Rename(kops.GetOutputDirectory(), outputDir)
	if err != nil {
		return errors.Wrap(err, "failed to rename kops output directory to tmp")
	}

	terraform := terraform.New(outputDir, logger)
	defer terraform.Close()
	err = terraform.Init()
	if err != nil {
		return err
	}

	err = terraform.Apply()
	if err != nil {
		return err
	}

	logger.WithField("dns", dns).Info("successfully created cluster")

	return nil
}

// DeleteCluster deletes a previously created cluster using kops and terraform.
func DeleteCluster(clusterId string, logger log.FieldLogger) error {
	logger = logger.WithField("cluster", clusterId)

	logger.Info("deleting cluster")

	s3StateStore := "dev.cloud.mattermost.com"
	dns := fmt.Sprintf("%s-kops.k8s.local", clusterId)

	// Temporarily look for the kops output directory as a local folder named `tmp`. See above.
	outputDir := "tmp"

	terraform := terraform.New(outputDir, logger)
	defer terraform.Close()
	err := terraform.Init()
	if err != nil {
		return err
	}
	err = terraform.Destroy()
	if err != nil {
		return err
	}

	kops, err := kops.New(s3StateStore, logger)
	defer kops.Close()
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}

	err = kops.DeleteCluster(dns)
	if err != nil {
		return errors.Wrap(err, "failed to delete cluster")
	}

	err = os.RemoveAll(outputDir)
	if err != nil {
		return errors.Wrap(err, "failed to clean up output directory")
	}

	logger.Info("successfully deleted cluster")

	return nil
}
