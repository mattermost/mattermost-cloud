package provisioner

import (
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"

	log "github.com/sirupsen/logrus"
)

// A Utility is a service that runs one per cluster but is not part of
// k8s  itself,  nor  is  it  part  of  a  ClusterInstallation  or  an
// Installation
type Utility interface {
	Create() error
	Upgrade() error
	Destroy() error
}

// utilityGroup  holds  the  metadata  needed  to  manage  a  specific
// utilityGroup,  and therefore  uniquely identifies  one, and  can be
// thought  of as  a handle  to the  real group  of utilities  running
// inside of the cluster
type utilityGroup struct {
	utilities   []Utility
	kops        *kops.Cmd
	provisioner *KopsProvisioner
}

func newUtilityGroupHandle(kops *kops.Cmd, provisioner *KopsProvisioner, cluster *model.Cluster, awsClient aws.AWS, parentLogger log.FieldLogger) (*utilityGroup, error) {
	logger := parentLogger.WithField("utility-group", "create-handle")

	desiredVersion, err := cluster.UtilityVersion("nginx")
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

	desiredVersion, err = cluster.UtilityVersion("fluentbit")
	if err != nil {
		return nil, err
	}

	fluentbit, err := newFluentbitHandle(desiredVersion, provisioner, awsClient, kops, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for Fluentbit")
	}

	return &utilityGroup{
		utilities:   []Utility{nginx, prometheus, fluentbit},
		kops:        kops,
		provisioner: provisioner,
	}, nil

}

// CreateUtilityGroup  creates  and  starts  all of  the  third  party
// services needed to run a cluster.
func (group utilityGroup) CreateUtilityGroup() error {
	// TODO remove this when Helm is removed as a dependency
	err := installHelm(group.kops, group.provisioner.logger)
	if err != nil {
		return errors.Wrap(err, "failed to set up Helm as a prerequisite to installing the cluster utilities")
	}

	for _, utility := range group.utilities {
		err := utility.Create()
		if err != nil {
			return errors.Wrap(err, "failed to provision one of the cluster utilities")
		}
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
	}

	return nil
}

// UpgradeUtilityGroup reapplies the chart for the UtilityGroup. This will cause services to upgrade to a new version, if one is available.
func (group utilityGroup) UpgradeUtilityGroup() error {
	for _, utility := range group.utilities {
		err := utility.Upgrade()
		if err != nil {
			return errors.Wrap(err, "failed to upgrade one of the cluster utilities")
		}
	}

	return nil
}
