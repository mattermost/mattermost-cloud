// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type thanos struct {
	awsClient      aws.AWS
	cluster        *model.Cluster
	kops           *kops.Cmd
	logger         log.FieldLogger
	provisioner    *KopsProvisioner
	actualVersion  *model.HelmUtilityVersion
	desiredVersion *model.HelmUtilityVersion
}

func newThanosHandle(cluster *model.Cluster, provisioner *KopsProvisioner, awsClient aws.AWS, kops *kops.Cmd, logger log.FieldLogger) (*thanos, error) {
	if logger == nil {
		return nil, fmt.Errorf("cannot instantiate Thanos handle with nil logger")
	}

	if cluster == nil {
		return nil, errors.New("cannot create a connection to Thanos if the cluster provided is nil")
	}

	if provisioner == nil {
		return nil, errors.New("cannot create a connection to Thanos if the provisioner provided is nil")
	}

	if awsClient == nil {
		return nil, errors.New("cannot create a connection to Thanos if the awsClient provided is nil")
	}

	if kops == nil {
		return nil, errors.New("cannot create a connection to Thanos if the Kops command provided is nil")
	}

	version := cluster.DesiredUtilityVersion(model.ThanosCanonicalName)

	return &thanos{
		awsClient:      awsClient,
		cluster:        cluster,
		kops:           kops,
		logger:         logger.WithField("cluster-utility", model.ThanosCanonicalName),
		provisioner:    provisioner,
		desiredVersion: version,
	}, nil
}

func (t *thanos) ValuesPath() string {
	if t.desiredVersion == nil {
		return ""
	}
	return t.desiredVersion.Values()
}

func (t *thanos) CreateOrUpgrade() error {
	logger := t.logger.WithField("thanos-action", "create")

	environment, err := t.awsClient.GetCloudEnvironmentName()
	if err != nil {
		return errors.Wrap(err, "failed to get environment name for thanos objstore secret")
	}

	if environment == "" {
		return errors.New("cannot create a thanos objstore secret if environment is empty")
	}

	awsRegion := os.Getenv("AWS_REGION")
	if awsRegion == "" {
		awsRegion = aws.DefaultAWSRegion
	}

	privateDomainName, err := t.awsClient.GetPrivateZoneDomainName(logger)
	if err != nil {
		return errors.Wrap(err, "unable to lookup private zone name")
	}

	app := "thanos"
	dns := fmt.Sprintf("%s.%s.%s", t.cluster.ID, app, privateDomainName)
	grpcDNS := fmt.Sprintf("%s-grpc.%s.%s", t.cluster.ID, app, privateDomainName)

	h := t.NewHelmDeployment(dns, grpcDNS)

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

		endpoint, err := getPrivateLoadBalancerEndpoint(ctx, "nginx", logger.WithField("thanos-action", "create"), t.kops.GetKubeConfigPath())
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

		endpoint, err := getPrivateLoadBalancerEndpoint(ctx, "prometheus", logger.WithField("thanos-action", "create"), t.kops.GetKubeConfigPath())
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

	logger.Infof("Deleting Route53 DNS Record for %s", app)
	err = t.awsClient.DeletePrivateCNAME(dns, logger.WithField("thanos-dns-delete", dns))
	if err != nil {
		return errors.Wrap(err, "failed to delete Route53 DNS record")
	}

	grpcDNS := fmt.Sprintf("%s-grpc.%s.%s", t.cluster.ID, app, privateDomainName)
	logger.Infof("Deleting GRPC Route53 DNS Record for %s", app)
	err = t.awsClient.DeletePrivateCNAME(grpcDNS, logger.WithField("thanos-dns-delete", grpcDNS))
	if err != nil {
		return errors.Wrap(err, "failed to delete GRPC Route53 DNS record")
	}

	t.actualVersion = nil
	return nil
}

func (t *thanos) Migrate() error {
	return nil
}

func (t *thanos) NewHelmDeployment(thanosDNS, thanosDNSGRPC string) *helmDeployment {
	helmValueArguments := fmt.Sprintf("query.ingress.hostname=%s,query.ingress.grpc.hostname=%s,query.ingress.annotations.nginx\\.ingress\\.kubernetes\\.io/whitelist-source-range=%s", thanosDNS, thanosDNSGRPC, strings.Join(t.provisioner.allowCIDRRangeList, "\\,"))

	return &helmDeployment{
		chartDeploymentName: "thanos",
		chartName:           "bitnami/thanos",
		kops:                t.kops,
		kopsProvisioner:     t.provisioner,
		logger:              t.logger,
		namespace:           "prometheus",
		setArgument:         helmValueArguments,
		desiredVersion:      t.desiredVersion,
	}
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
