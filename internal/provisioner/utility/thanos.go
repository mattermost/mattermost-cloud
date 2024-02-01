// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utility

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/provisioner/prometheus"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type thanos struct {
	awsClient          aws.AWS
	kubeconfigPath     string
	allowCIDRRangeList []string
	cluster            *model.Cluster
	logger             log.FieldLogger
	actualVersion      *model.HelmUtilityVersion
	desiredVersion     *model.HelmUtilityVersion
}

func newThanosOrUnmanagedHandle(cluster *model.Cluster, kubeconfigPath string, allowCIDRRangeList []string, awsClient aws.AWS, logger log.FieldLogger) (Utility, error) {
	desired := cluster.DesiredUtilityVersion(model.ThanosCanonicalName)
	actual := cluster.ActualUtilityVersion(model.ThanosCanonicalName)

	if model.UtilityIsUnmanaged(desired, actual) {
		return newUnmanagedHandle(model.ThanosCanonicalName, logger), nil
	}
	thanos := newThanosHandle(cluster, desired, kubeconfigPath, allowCIDRRangeList, awsClient, logger)
	err := thanos.validate()
	if err != nil {
		return nil, errors.Wrap(err, "teleport utility config is invalid")
	}

	return thanos, nil
}

func newThanosHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, kubeconfigPath string, allowCIDRRangeList []string, awsClient aws.AWS, logger log.FieldLogger) *thanos {
	return &thanos{
		awsClient:          awsClient,
		kubeconfigPath:     kubeconfigPath,
		allowCIDRRangeList: allowCIDRRangeList,
		cluster:            cluster,
		logger:             logger.WithField("cluster-utility", model.ThanosCanonicalName),
		desiredVersion:     desiredVersion,
		actualVersion:      cluster.UtilityMetadata.ActualVersions.Thanos,
	}
}

func (t *thanos) validate() error {
	if t.kubeconfigPath == "" {
		return errors.New("kubeconfig path cannot be empty")
	}
	if t.awsClient == nil {
		return errors.New("awsClient cannot be nil")
	}

	return nil
}

func (t *thanos) ValuesPath() string {
	if t.desiredVersion == nil {
		return ""
	}
	return t.desiredVersion.Values()
}

func (t *thanos) CreateOrUpgrade() error {
	logger := t.logger.WithField("thanos-action", "create")

	privateDomainName, err := t.awsClient.GetPrivateZoneDomainName(logger)
	if err != nil {
		return errors.Wrap(err, "unable to lookup private zone name")
	}

	app := "thanos"
	dns := fmt.Sprintf("%s.%s.%s", t.cluster.ID, app, privateDomainName)
	grpcDNS := fmt.Sprintf("%s-grpc.%s.%s", t.cluster.ID, app, privateDomainName)

	h := t.newHelmDeployment(dns)

	err = h.Update()
	if err != nil {
		return errors.Wrap(err, "failed to create the Thanos Helm deployment")
	}

	err = t.updateVersion(h)
	if err != nil {
		return err
	}

	if t.awsClient.IsProvisionedPrivateCNAME(dns, t.logger) {
		t.logger.Debugln("Main CNAME was already provisioned for thanos")
	} else {
		t.logger.Debugln("Main CNAME was not provisioned for thanos")
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(120)*time.Second)
		defer cancel()

		endpoint, err := getPrivateLoadBalancerEndpoint(ctx, namespaceNginxInternal, logger.WithField("thanos-action", "create"), t.kubeconfigPath)
		if err != nil {
			return errors.Wrap(err, "couldn't get the load balancer endpoint (nginx) for Thanos")
		}

		logger.Infof("Registering DNS %s for %s", dns, app)
		err = t.awsClient.CreatePrivateCNAME(dns, []string{endpoint}, logger.WithField("thanos-dns-create", dns))
		if err != nil {
			return errors.Wrap(err, "failed to create a CNAME to point to Thanos")
		}
	}

	if t.awsClient.IsProvisionedPrivateCNAME(grpcDNS, t.logger) {
		t.logger.Debugln("GRPC CNAME was already provisioned for thanos")
	} else {
		t.logger.Debugln("GRPC CNAME was not provisioned for thanos")
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(120)*time.Second)
		defer cancel()

		endpoint, err := getPrivateLoadBalancerEndpoint(ctx, prometheus.Namespace, logger.WithField("thanos-action", "create"), t.kubeconfigPath)
		if err != nil {
			return errors.Wrap(err, "couldn't get the load balancer endpoint for Thanos")
		}

		logger.Infof("Registering GRPC DNS %s for %s", grpcDNS, app)
		err = t.awsClient.CreatePrivateCNAME(grpcDNS, []string{endpoint}, logger.WithField("thanos-dns-create", grpcDNS))
		if err != nil {
			return errors.Wrap(err, "failed to create a CNAME to point to Thanos GRPC")
		}
	}
	return nil
}

func (t *thanos) Destroy() error {
	logger := t.logger.WithField("prometheus-action", "destroy")

	privateDomainName, err := t.awsClient.GetPrivateZoneDomainName(logger)
	if err != nil {
		return errors.Wrap(err, "unable to lookup private zone name")
	}
	app := "thanos"
	dns := fmt.Sprintf("%s.%s.%s", t.cluster.ID, app, privateDomainName)

	logger.WithField("dns", dns).Infof("Deleting Route53 DNS Record for %s", app)
	err = t.awsClient.DeletePrivateCNAME(dns, logger.WithField("thanos-dns-delete", dns))
	if err != nil {
		return errors.Wrap(err, "failed to delete Route53 DNS record")
	}

	grpcDNS := fmt.Sprintf("%s-grpc.%s.%s", t.cluster.ID, app, privateDomainName)
	logger.WithField("grpcDNS", grpcDNS).Infof("Deleting GRPC Route53 DNS Record for %s", app)
	err = t.awsClient.DeletePrivateCNAME(grpcDNS, logger.WithField("thanos-dns-delete", grpcDNS))
	if err != nil {
		return errors.Wrap(err, "failed to delete GRPC Route53 DNS record")
	}

	t.actualVersion = nil

	helm := t.newHelmDeployment(dns)
	return helm.Delete()
}

func (t *thanos) Migrate() error {
	return nil
}

func (t *thanos) newHelmDeployment(thanosDNS string) *helmDeployment {
	helmValueArguments := fmt.Sprintf("query.ingress.hostname=%s,query.ingress.annotations.nginx\\.ingress\\.kubernetes\\.io/whitelist-source-range=%s", thanosDNS, strings.Join(t.allowCIDRRangeList, "\\,"))

	return newHelmDeployment(
		"bitnami/thanos",
		"thanos",
		prometheus.Namespace,
		t.kubeconfigPath,
		t.desiredVersion,
		helmValueArguments,
		t.logger,
	)
}

func (t *thanos) Name() string {
	return model.ThanosCanonicalName
}

func (t *thanos) DesiredVersion() *model.HelmUtilityVersion {
	return t.desiredVersion
}

func (t *thanos) ActualVersion() *model.HelmUtilityVersion {
	if t.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(t.actualVersion.Version(), "thanos-"),
		ValuesPath: t.actualVersion.Values(),
	}
}

func (t *thanos) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	t.actualVersion = actualVersion
	return nil
}
