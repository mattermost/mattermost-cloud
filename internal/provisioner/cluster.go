package provisioner

import (
	"fmt"
	"os"
	"path"

	"github.com/mattermost/mattermost-server/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/internal/tools/terraform"
)

// clusterRootDir is the local directory that contains cluster configuration.
const clusterRootDir = "clusters"

// CreateCluster creates a cluster using kops and terraform.
func CreateCluster(provider, s3StateStore, size string, zones []string, logger log.FieldLogger) error {
	provider, err := checkProvider(provider)
	if err != nil {
		return err
	}

	kopsClusterSize, err := kops.GetSize(size)
	if err != nil {
		return err
	}

	clusterId := model.NewId()

	// Temporarily locate the kops output directory to a local folder based on the
	// cluster name. This won't be necessary once we persist the output to S3 instead.
	_, err = os.Stat(clusterRootDir)
	if err != nil && os.IsNotExist(err) {
		err = os.Mkdir(clusterRootDir, 0755)
		if err != nil {
			return errors.Wrap(err, "unable to create cluster root dir")
		}
	} else if err != nil {
		return errors.Wrapf(err, "failed to stat cluster root directory %q", clusterRootDir)
	}

	outputDir := path.Join(clusterRootDir, clusterId)
	_, err = os.Stat(outputDir)
	if err == nil {
		return fmt.Errorf("encountered cluster ID collision: directory %q already exists", outputDir)
	} else if err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to stat cluster directory %q", outputDir)
	}

	dns := clusterDNS(clusterId)

	logger = logger.WithField("cluster", clusterId)

	logger.WithField("dns", dns).Info("creating cluster")

	kops, err := kops.New(s3StateStore, logger)
	if err != nil {
		return err
	}
	defer kops.Close()
	err = kops.CreateCluster(dns, provider, kopsClusterSize, zones)
	if err != nil {
		return err
	}

	err = os.Rename(kops.GetOutputDirectory(), outputDir)
	if err != nil {
		return fmt.Errorf("failed to rename kops output directory to %q", outputDir)
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
func DeleteCluster(clusterId, s3StateStore string, logger log.FieldLogger) error {
	logger = logger.WithField("cluster", clusterId)

	dns := clusterDNS(clusterId)

	// Temporarily look for the kops output directory as a local folder named after
	// the cluster ID. See above.
	outputDir := path.Join(clusterRootDir, clusterId)

	// Validate the provided cluster ID before we alter state in any way.
	_, err := os.Stat(outputDir)
	if err != nil {
		return errors.Wrapf(err, "failed to find cluster directory %q", outputDir)
	}

	terraform := terraform.New(outputDir, logger)
	defer terraform.Close()
	err = terraform.Init()
	if err != nil {
		return err
	}
	out, err := terraform.Output("cluster_name")
	if err != nil {
		return err
	}
	if out != dns {
		return fmt.Errorf("terraform cluster_name (%s) does not match dns from provided ID (%s)", out, dns)
	}

	kops, err := kops.New(s3StateStore, logger)
	defer kops.Close()
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}
	_, err = kops.GetCluster(dns)
	if err != nil {
		return err
	}

	logger.Info("deleting cluster")
	err = terraform.Destroy()
	if err != nil {
		return err
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

func clusterDNS(id string) string {
	return fmt.Sprintf("%s-kops.k8s.local", id)
}
