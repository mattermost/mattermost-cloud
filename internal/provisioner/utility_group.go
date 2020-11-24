// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

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
	// CreateOrUpgrade is responsible for deploying the utility in the
	// cluster and then for updating it if it already exists when called
	CreateOrUpgrade() error

	// Destroy can be used if special care must be taken for deleting a
	// utility from a cluster
	Destroy() error

	// Migrate can be used if special care must be taken for migrating a
	// utility from a cluster
	Migrate() error

	// ActualVersion returns the utility's last reported actual version,
	// at the time of Create or Upgrade. This version will remain valid
	// unless something interacts with the cluster out of band, at which
	// time it will be invalid until Upgrade is called again
	ActualVersion() model.UtilityVersion

	// DesiredVersion returns the utility's target version, which has been
	// requested, but may not yet have been reconciled
	DesiredVersion() model.UtilityVersion

	// Name returns the canonical string-version name for the utility,
	// used throughout the application
	Name() string

	// ValuesPath returns the location where the values file(s) are
	// stored for this utility
	ValuesPath() string
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

// List of repos to add during helm setup
var helmRepos = map[string]string{
	"stable":               "https://charts.helm.sh/stable",
	"chartmuseum":          "https://chartmuseum.internal.core.cloud.mattermost.com",
	"ingress-nginx":        "https://kubernetes.github.io/ingress-nginx",
	"prometheus-community": "https://prometheus-community.github.io/helm-charts",
	"bitnami":              "https://charts.bitnami.com/bitnami",
}

func newUtilityGroupHandle(kops *kops.Cmd, provisioner *KopsProvisioner, cluster *model.Cluster, awsClient aws.AWS, parentLogger log.FieldLogger) (*utilityGroup, error) {
	logger := parentLogger.WithField("utility-group", "create-handle")

	nginx, err := newNginxHandle(
		cluster.DesiredUtilityVersion(model.NginxCanonicalName),
		provisioner, awsClient, kops, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for NGINX")
	}

	prometheusOperator, err := newPrometheusOperatorHandle(cluster, provisioner, awsClient, kops, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for Prometheus Operator")
	}

	thanos, err := newThanosHandle(cluster, provisioner, awsClient, kops, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for Thanos")
	}

	fluentbit, err := newFluentbitHandle(
		cluster.DesiredUtilityVersion(model.FluentbitCanonicalName),
		provisioner, awsClient, kops, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for Fluentbit")
	}

	teleport, err := newTeleportHandle(
		cluster, cluster.DesiredUtilityVersion(model.TeleportCanonicalName),
		provisioner, awsClient, kops, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for Teleport")
	}

	// the order of utilities here matters; the utilities are deployed
	// in order to resolve dependencies between them
	return &utilityGroup{
		utilities:   []Utility{nginx, prometheusOperator, thanos, fluentbit, teleport},
		kops:        kops,
		provisioner: provisioner,
		cluster:     cluster,
	}, nil

}

// CreateUtilityGroup  creates  and  starts  all of  the  third  party
// services needed to run a cluster.
func (group utilityGroup) CreateUtilityGroup() error {
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

// ProvisionUtilityGroup reapplies the chart for the UtilityGroup. This will cause services to upgrade to a new version, if one is available.
func (group utilityGroup) ProvisionUtilityGroup() error {
	logger := group.provisioner.logger.WithField("utility-group", "UpgradeManifests")

	logger.Info("Adding new Helm repos.")
	for repoName, repoURL := range helmRepos {
		err := helmRepoAdd(repoName, repoURL, logger)
		if err != nil {
			return errors.Wrap(err, "unable to add helm repos")
		}
	}

	for _, utility := range group.utilities {
		err := utility.CreateOrUpgrade()
		if err != nil {
			return errors.Wrap(err, "failed to upgrade one of the cluster utilities")
		}

		err = group.cluster.SetUtilityActualVersion(utility.Name(), utility.ActualVersion())
		if err != nil {
			return err
		}
	}

	return nil
}
