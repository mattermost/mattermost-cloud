package provisioner

import (
	"fmt"
	"os"
	"path"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"

	"github.com/mattermost/mattermost-cloud/internal/store"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/internal/tools/terraform"
)

// clusterRootDir is the local directory that contains cluster configuration.
const clusterRootDir = "clusters"

// clusterStore abstracts the database operations required to manage clusters.
type clusterStore interface {
	GetCluster(string) (*store.Cluster, error)
	CreateCluster(*store.Cluster) error
	DeleteCluster(string) error
}

// CreateCluster creates a cluster using kops and terraform.
func CreateCluster(cs clusterStore, provider, s3StateStore, size string, zones []string, logger log.FieldLogger) error {
	provider, err := checkProvider(provider)
	if err != nil {
		return err
	}

	kopsClusterSize, err := kops.GetSize(size)
	if err != nil {
		return err
	}

	kopsCluster := NewKopsCluster(provider)
	err = cs.CreateCluster(&kopsCluster.Cluster)
	if err != nil {
		return err
	}

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

	outputDir := path.Join(clusterRootDir, kopsCluster.ID)
	_, err = os.Stat(outputDir)
	if err == nil {
		return fmt.Errorf("encountered cluster ID collision: directory %q already exists", outputDir)
	} else if err != nil && !os.IsNotExist(err) {
		return errors.Wrapf(err, "failed to stat cluster directory %q", outputDir)
	}

	logger = logger.WithField("cluster", kopsCluster.ID)

	logger.WithField("name", kopsCluster.KopsName()).Info("creating cluster")

	kops, err := kops.New(s3StateStore, logger)
	if err != nil {
		return err
	}
	defer kops.Close()
	err = kops.CreateCluster(kopsCluster.KopsName(), provider, kopsClusterSize, zones)
	if err != nil {
		return err
	}

	err = os.Rename(kops.GetOutputDirectory(), outputDir)
	if err != nil {
		return fmt.Errorf("failed to rename kops output directory to %q", outputDir)
	}

	terraformClient := terraform.New(outputDir, logger)
	defer terraformClient.Close()
	err = terraformClient.Init()
	if err != nil {
		return err
	}

	err = terraformClient.ApplyTarget(fmt.Sprintf("aws_internet_gateway.%s-kops-k8s-local", kopsCluster.ID))
	if err != nil {
		return err
	}

	err = terraformClient.ApplyTarget(fmt.Sprintf("aws_elb.api-%s-kops-k8s-local", kopsCluster.ID))
	if err != nil {
		return err
	}

	err = kops.UpdateCluster(kopsCluster.KopsName())
	if err != nil {
		return err
	}

	err = terraformClient.Apply()
	if err != nil {
		return err
	}

	logger.WithField("name", kopsCluster.KopsName()).Info("successfully created cluster")

	return nil
}

// UpgradeCluster upgrades a cluster to the latest recommended production ready k8s version.
func UpgradeCluster(cs clusterStore, clusterID, s3StateStore string, logger log.FieldLogger) error {
	cluster, err := cs.GetCluster(clusterID)
	if err != nil {
		return err
	}
	if cluster == nil {
		return errors.Errorf("unknown cluster %s", clusterID)
	}

	kopsCluster := KopsClusterFromCluster(cluster)

	logger = logger.WithField("cluster", kopsCluster.ID)

	// Temporarily look for the kops output directory as a local folder named after
	// the cluster ID. See above.
	outputDir := path.Join(clusterRootDir, kopsCluster.ID)

	// Validate the provided cluster ID before we alter state in any way.
	_, err = os.Stat(outputDir)
	if err != nil {
		return errors.Wrapf(err, "failed to find cluster directory %q", outputDir)
	}

	terraformClient := terraform.New(outputDir, logger)
	defer terraformClient.Close()
	err = terraformClient.Init()
	if err != nil {
		return err
	}
	out, err := terraformClient.Output("cluster_name")
	if err != nil {
		return err
	}
	if out != kopsCluster.KopsName() {
		return fmt.Errorf("terraform cluster_name (%s) does not match kops name from provided ID (%s)", out, kopsCluster.KopsName())
	}

	kops, err := kops.New(s3StateStore, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()
	_, err = kops.GetCluster(kopsCluster.KopsName())
	if err != nil {
		return err
	}

	logger.Info("upgrading cluster")

	err = kops.UpgradeCluster(kopsCluster.KopsName())
	if err != nil {
		return err
	}
	err = kops.UpdateCluster(kopsCluster.KopsName())
	if err != nil {
		return err
	}

	err = terraformClient.Apply()
	if err != nil {
		return err
	}

	err = kops.RollingUpdateCluster(kopsCluster.KopsName())
	if err != nil {
		return err
	}
	err = kops.ValidateCluster(kopsCluster.KopsName())
	if err != nil {
		return err
	}

	logger.Info("successfully upgraded cluster")

	return nil
}

// DeleteCluster deletes a previously created cluster using kops and terraform.
func DeleteCluster(cs clusterStore, clusterID, s3StateStore string, logger log.FieldLogger) error {
	cluster, err := cs.GetCluster(clusterID)
	if err != nil {
		return err
	}
	if cluster == nil {
		return errors.Errorf("unknown cluster %s", clusterID)
	}

	kopsCluster := KopsClusterFromCluster(cluster)

	logger = logger.WithField("cluster", kopsCluster.ID)

	// Temporarily look for the kops output directory as a local folder named after
	// the cluster ID. See above.
	outputDir := path.Join(clusterRootDir, kopsCluster.ID)

	// Validate the provided cluster ID before we alter state in any way.
	_, err = os.Stat(outputDir)
	if err != nil {
		return errors.Wrapf(err, "failed to find cluster directory %q", outputDir)
	}

	terraformClient := terraform.New(outputDir, logger)
	defer terraformClient.Close()
	err = terraformClient.Init()
	if err != nil {
		return err
	}
	out, err := terraformClient.Output("cluster_name")
	if err != nil {
		return err
	}
	if out != kopsCluster.KopsName() {
		return fmt.Errorf("terraform cluster_name (%s) does not match dns from provided ID (%s)", out, kopsCluster.KopsName())
	}

	kops, err := kops.New(s3StateStore, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()
	_, err = kops.GetCluster(kopsCluster.KopsName())
	if err != nil {
		return err
	}

	logger.Info("deleting cluster")
	err = terraformClient.Destroy()
	if err != nil {
		return err
	}

	err = kops.DeleteCluster(kopsCluster.KopsName())
	if err != nil {
		return errors.Wrap(err, "failed to delete cluster")
	}

	err = os.RemoveAll(outputDir)
	if err != nil {
		return errors.Wrap(err, "failed to clean up output directory")
	}

	err = cs.DeleteCluster(kopsCluster.ID)
	if err != nil {
		return err
	}

	logger.Info("successfully deleted cluster")

	return nil
}
