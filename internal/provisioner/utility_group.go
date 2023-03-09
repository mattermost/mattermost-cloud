// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"github.com/mattermost/mattermost-cloud/internal/provisioner/helm"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
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
	ActualVersion() *model.HelmUtilityVersion

	// DesiredVersion returns the utility's target version, which has been
	// requested, but may not yet have been reconciled
	DesiredVersion() *model.HelmUtilityVersion

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
	utilities []Utility
	logger    log.FieldLogger
	cluster   *model.Cluster
}

// List of repos to add during helm setup
var helmRepos = map[string]string{
	"chartmuseum":          "https://chartmuseum.internal.core.cloud.mattermost.com",
	"ingress-nginx":        "https://kubernetes.github.io/ingress-nginx",
	"prometheus-community": "https://prometheus-community.github.io/helm-charts",
	"bitnami":              "https://charts.bitnami.com/bitnami",
	"fluent":               "https://fluent.github.io/helm-charts",
	"grafana":              "https://grafana.github.io/helm-charts",
	"deliveryhero":         "https://charts.deliveryhero.io/",
	"metrics-server":       "https://kubernetes-sigs.github.io/metrics-server/",
	"vmware-tanzu":         "https://vmware-tanzu.github.io/helm-charts/",
	"mattermost":           "https://helm.mattermost.com",
}

func newUtilityGroupHandle(
	params ProvisioningParams,
	kubeconfigPath string,
	cluster *model.Cluster,
	awsClient aws.AWS,
	parentLogger log.FieldLogger,
) (*utilityGroup, error) {
	logger := parentLogger.WithField("utility-group", "create-handle")

	nginx, err := helm.NewNginxHandle(
		cluster.DesiredUtilityVersion(model.NginxCanonicalName),
		cluster, kubeconfigPath, awsClient, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for NGINX")
	}

	nginxInternal, err := helm.NewNginxInternalHandle(
		cluster.DesiredUtilityVersion(model.NginxInternalCanonicalName),
		cluster, kubeconfigPath, awsClient, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for NGINX INTERNAL")
	}

	prometheusOperator, err := helm.NewPrometheusOperatorHandle(cluster, kubeconfigPath, params.AllowCIDRRangeList, awsClient, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for Prometheus Operator")
	}

	thanos, err := helm.NewThanosHandle(cluster, kubeconfigPath, params.AllowCIDRRangeList, awsClient, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for Thanos")
	}

	fluentbit, err := helm.NewFluentbitHandle(
		cluster,
		cluster.DesiredUtilityVersion(model.FluentbitCanonicalName),
		kubeconfigPath, awsClient, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for Fluentbit")
	}

	teleport, err := helm.NewTeleportHandle(
		cluster, cluster.DesiredUtilityVersion(model.TeleportCanonicalName),
		kubeconfigPath, awsClient, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for Teleport")
	}

	pgbouncer, err := helm.NewPgbouncerHandle(
		cluster, cluster.DesiredUtilityVersion(model.PgbouncerCanonicalName),
		kubeconfigPath, awsClient, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for Pgbouncer")
	}

	promtail, err := helm.NewPromtailHandle(
		cluster, cluster.DesiredUtilityVersion(model.PromtailCanonicalName),
		kubeconfigPath, awsClient, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for Pgbouncer")
	}

	rtcd, err := helm.NewRtcdHandle(
		cluster, cluster.DesiredUtilityVersion(model.RtcdCanonicalName),
		kubeconfigPath, awsClient, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for RTCD")
	}

	nodeProblemDetector, err := helm.NewNodeProblemDetectorHandle(
		cluster.DesiredUtilityVersion(model.NodeProblemDetectorCanonicalName),
		cluster, kubeconfigPath, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for Node Problem Detector")
	}

	metricsServer, err := helm.NewMetricsServerHandle(
		cluster.DesiredUtilityVersion(model.MetricsServerCanonicalName),
		cluster, kubeconfigPath, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for Metrics Server")
	}

	velero, err := helm.NewVeleroHandle(
		cluster.DesiredUtilityVersion(model.VeleroCanonicalName), cluster,
		kubeconfigPath, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for Velero")
	}

	cloudprober, err := helm.NewCloudproberHandle(
		cluster.DesiredUtilityVersion(model.CloudproberCanonicalName), cluster,
		kubeconfigPath, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get handle for cloudprober")
	}

	// the order of utilities here matters; the utilities are deployed
	// in order to resolve dependencies between them
	return &utilityGroup{
		utilities: []Utility{nginx, nginxInternal, prometheusOperator, thanos, fluentbit, teleport, pgbouncer, promtail, nodeProblemDetector, rtcd, metricsServer, velero, cloudprober},
		logger:    logger,
		cluster:   cluster,
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
			group.logger.WithError(err).Warnf("failed to destroy utility `%s`", utility.Name())
		}
	}

	return nil
}

// ProvisionUtilityGroup reapplies the chart for the UtilityGroup. This will cause services to upgrade to a new version, if one is available.
func (group utilityGroup) ProvisionUtilityGroup() error {
	logger := group.logger.WithField("utility-group", "UpgradeManifests")

	logger.Info("Adding new Helm repos.")
	for repoName, repoURL := range helmRepos {
		err := helm.AddRepo(repoName, repoURL, logger)
		if err != nil {
			return errors.Wrap(err, "unable to add helm repos")
		}
	}

	for _, utility := range group.utilities {
		if utility.DesiredVersion().IsEmpty() {
			logger.Infof("Skipping reprovision of utility \"%s\"", utility.Name())
		} else {
			err := utility.CreateOrUpgrade()
			if err != nil {
				return errors.Wrap(err, "failed to upgrade one of the cluster utilities")
			}
		}

		err := group.cluster.SetUtilityActualVersion(utility.Name(), utility.ActualVersion())
		if err != nil {
			return err
		}
	}

	return nil
}
