package provisioner

import (
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// A Utility is a service that runs one per cluster but is not part of
// k8s itself, nor is it part of a ClusterInstallation or an
// Installation
type Utility interface {
	// Create is responsible for deploying the utility in the cluster
	Create() error

	// Upgrade is responsible for handling changes to an existing
	// utility
	Upgrade() error

	// Destroy can be used if special care must be taken for deleting a
	// utility from a cluster
	Destroy() error

	// ActualVersion returns the utility's last reported actual version,
	// at the time of Create or Upgrade. This version will remain valid
	// unless something interacts with the cluster out of band, at which
	// time it will be invalid until Upgrade is called again
	ActualVersion() string

	// DesiredVersion returns the utility's target version, which has been
	// requested, but may not yet have been reconciled
	DesiredVersion() string

	// Name returns the canonical string-version name for the utility,
	// used throughout the application
	Name() string
}

// utilityGroup  holds  the  metadata  needed  to  manage  a  specific
// utilityGroup,  and therefore  uniquely identifies  one, and  can be
// thought  of as  a handle  to the  real group  of utilities  running
// inside of the cluster
type utilityGroup struct {
	utilities   []Utility
	kops        *kops.Cmd
	provisioner *KopsProvisioner
	cluster     *model.Cluster
}

func newUtilityGroupHandle(kops *kops.Cmd, provisioner *KopsProvisioner, cluster *model.Cluster, awsClient aws.AWS, parentLogger log.FieldLogger) (*utilityGroup, error) {
	logger := parentLogger.WithField("utility-group", "create-handle")

	desiredVersion, err := cluster.DesiredUtilityVersion(model.NginxCanonicalName)
	if err != nil {
		return nil, err
	}

	nginx, err := newNginxHandle(desiredVersion, provisioner, kops, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for NGINX")
	}

	prometheus, err := newPrometheusHandle(cluster, provisioner, awsClient, kops, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for Prometheus")
	}

	desiredVersion, err = cluster.DesiredUtilityVersion(model.FluentbitCanonicalName)
	if err != nil {
		return nil, err
	}

	fluentbit, err := newFluentbitHandle(desiredVersion, provisioner, awsClient, kops, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for Fluentbit")
	}

	desiredVersion, err = cluster.DesiredUtilityVersion(model.CertManagerCanonicalName)
	if err != nil {
		return nil, err
	}

	certManager, err := newCertManagerHandle(desiredVersion, provisioner, kops, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for Cert Manager")
	}

	desiredVersion, err = cluster.DesiredUtilityVersion(model.PublicNginxCanonicalName)
	if err != nil {
		return nil, err
	}

	publicNginx, err := newPublicNginxHandle(desiredVersion, provisioner, kops, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for Cert Manager")
	}

	// the order of utilities here matters; the utilities are deployed
	// in order to resolve dependencies between them
	return &utilityGroup{
		utilities:   []Utility{nginx, publicNginx, prometheus, fluentbit, certManager},
		kops:        kops,
		provisioner: provisioner,
		cluster:     cluster,
	}, nil

}

// CreateUtilityGroup  creates  and  starts  all of  the  third  party
// services needed to run a cluster.
func (group utilityGroup) CreateUtilityGroup() error {
	logger := group.provisioner.logger.WithField("utility-group", "DeployManifests")

	err := deployCertManagerCRDS(group.kops, logger)
	if err != nil {
		return errors.Wrap(err, "failed to deploy Cert Manager manifests")
	}

	// List of repos to add during helm setup
	var helmRepos = map[string]string{
		"jetstack": "https://charts.jetstack.io",
		"stable":   "https://kubernetes-charts.storage.googleapis.com",
	}

	// TODO remove this when Helm is removed as a dependency
	err = installHelm(group.kops, helmRepos, group.provisioner.logger)
	if err != nil {
		return errors.Wrap(err, "failed to set up Helm as a prerequisite to installing the cluster utilities")
	}

	for _, utility := range group.utilities {
		err := utility.Create()
		if err != nil {
			return errors.Wrap(err, "failed to provision one of the cluster utilities")
		}

		err = group.cluster.SetUtilityActualVersion(utility.Name(), utility.ActualVersion())
		if err != nil {
			return err
		}
	}

	err = deployClusterIssuer(group.kops, logger)
	if err != nil {
		return errors.Wrap(err, "failed to deploy ClusterIssuer manifest")
	}

	return nil
}

// DestroyUtilityGroup tears down all of the third party services in a
// UtilityGroup
func (group utilityGroup) DestroyUtilityGroup() error {
	for _, utility := range group.utilities {
		err := utility.Destroy()
		if err != nil {
			return errors.Wrap(err, "failed to destroy one of the cluster utilities")
		}

		err = group.cluster.SetUtilityActualVersion(utility.Name(), utility.ActualVersion())
		if err != nil {
			return err
		}
	}

	return nil
}

// UpgradeUtilityGroup reapplies the chart for the UtilityGroup. This will cause services to upgrade to a new version, if one is available.
func (group utilityGroup) UpgradeUtilityGroup() error {
	logger := group.provisioner.logger.WithField("utility-group", "UpgradeManifests")

	err := deployCertManagerCRDS(group.kops, logger)
	if err != nil {
		return errors.Wrap(err, "failed to deploy Cert Manager manifests")
	}

	logger = group.provisioner.logger.WithField("helm-init", "UpgradeUtilityGroup")
	err = helmInit(logger, group.kops)
	if err != nil {
		logger.WithError(err).Error("couldn't re-initialize Helm in the cluster")
	}

	for _, utility := range group.utilities {
		err := utility.Upgrade()
		if err != nil {
			return errors.Wrap(err, "failed to upgrade one of the cluster utilities")
		}

		err = group.cluster.SetUtilityActualVersion(utility.Name(), utility.ActualVersion())
		if err != nil {
			return err
		}
	}

	err = deployClusterIssuer(group.kops, logger)
	if err != nil {
		return errors.Wrap(err, "failed to deploy ClusterIssuer manifest")
	}

	return nil
}
