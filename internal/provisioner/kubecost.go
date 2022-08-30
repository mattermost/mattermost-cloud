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

	"github.com/mattermost/mattermost-cloud/k8s"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

type kubecost struct {
	awsClient          aws.AWS
	cluster            *model.Cluster
	kubeconfigPath     string
	allowCIDRRangeList []string
	logger             log.FieldLogger
	desiredVersion     *model.HelmUtilityVersion
	actualVersion      *model.HelmUtilityVersion
}

func newKubecostHandle(cluster *model.Cluster, desiredVersion *model.HelmUtilityVersion, kubeconfigPath string, allowCIDRRangeList []string, awsClient aws.AWS, logger log.FieldLogger) (*kubecost, error) {
	if logger == nil {
		return nil, errors.New("cannot instantiate Kubecost handle with nil logger")
	}

	if cluster == nil {
		return nil, errors.New("cannot create a connection to Kubecost if the cluster provided is nil")
	}
	if awsClient == nil {
		return nil, errors.New("cannot create a connection to Kubecost if the awsClient provided is nil")
	}
	if kubeconfigPath == "" {
		return nil, errors.New("cannot create utility without kubeconfig")
	}

	return &kubecost{
		awsClient:          awsClient,
		cluster:            cluster,
		kubeconfigPath:     kubeconfigPath,
		allowCIDRRangeList: allowCIDRRangeList,
		logger:             logger.WithField("cluster-utility", model.KubecostCanonicalName),
		desiredVersion:     desiredVersion,
		actualVersion:      cluster.UtilityMetadata.ActualVersions.Kubecost,
	}, nil

}

func (k *kubecost) updateVersion(h *helmDeployment) error {
	actualVersion, err := h.Version()
	if err != nil {
		return err
	}

	k.actualVersion = actualVersion
	return nil
}

func (k *kubecost) ValuesPath() string {
	if k.desiredVersion == nil {
		return ""
	}
	return k.desiredVersion.Values()
}

func (k *kubecost) CreateOrUpgrade() error {
	logger := k.logger.WithField("kubecost-action", "create")

	k8sClient, err := k8s.NewFromFile(k.kubeconfigPath, logger)
	if err != nil {
		return errors.Wrap(err, "failed to set up the k8s client")
	}
	_, err = k8sClient.CreateOrUpdateNamespace("kubecost")
	if err != nil {
		return errors.Wrapf(err, "failed to create the kubecost namespace")
	}

	privateDomainName, err := k.awsClient.GetPrivateZoneDomainName(logger)
	if err != nil {
		return errors.Wrap(err, "unable to lookup private zone name")
	}

	app := "kubecost"
	dns := fmt.Sprintf("%s.%s.%s", k.cluster.ID, app, privateDomainName)

	h := k.NewHelmDeployment(dns)

	err = h.Update()
	if err != nil {
		return err
	}

	err = k.updateVersion(h)
	if err != nil {
		return err
	}

	if k.awsClient.IsProvisionedPrivateCNAME(dns, k.logger) {
		k.logger.Debugln("CNAME was already provisioned for kubecost")
		return nil
	}
	k.logger.Debugln("CNAME was not provisioned for kubecost")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(180)*time.Second)
	defer cancel()

	endpoint, err := getPrivateLoadBalancerEndpoint(ctx, "nginx-internal", logger, k.kubeconfigPath)
	if err != nil {
		return errors.Wrap(err, "couldn't get the load balancer endpoint (nginx) for Prometheus")
	}

	logger.Infof("Registering DNS %s for %s", dns, app)
	err = k.awsClient.CreatePrivateCNAME(dns, []string{endpoint}, logger.WithField("kubecost-dns-create", dns))
	if err != nil {
		return errors.Wrap(err, "failed to create a CNAME to point to Kubecost")
	}

	return nil
}

func (k *kubecost) DesiredVersion() *model.HelmUtilityVersion {
	return k.desiredVersion
}

func (k *kubecost) ActualVersion() *model.HelmUtilityVersion {
	if k.actualVersion == nil {
		return nil
	}
	return &model.HelmUtilityVersion{
		Chart:      strings.TrimPrefix(k.actualVersion.Version(), "cost-analyzer-"),
		ValuesPath: k.actualVersion.Values(),
	}
}

func (k *kubecost) Destroy() error {
	logger := k.logger.WithField("kubecost-action", "destroy")

	privateDomainName, err := k.awsClient.GetPrivateZoneDomainName(logger)
	if err != nil {
		return errors.Wrap(err, "unable to lookup private zone name")
	}
	app := "kubecost"
	dns := fmt.Sprintf("%s.%s.%s", k.cluster.ID, app, privateDomainName)

	logger.Infof("Deleting Route53 DNS Record for %s", app)
	err = k.awsClient.DeletePrivateCNAME(dns, logger.WithField("kubecost-dns-delete", dns))
	if err != nil {
		return errors.Wrap(err, "failed to delete Route53 DNS record")
	}

	k.actualVersion = nil
	return nil
}

func (k *kubecost) Migrate() error {
	return nil
}

func (k *kubecost) NewHelmDeployment(kubecostDNS string) *helmDeployment {
	kubecostToken := ""
	if len(os.Getenv(model.KubecostToken)) > 0 {
		kubecostToken = os.Getenv(model.KubecostToken)
	}
	helmValueArguments := fmt.Sprintf("kubecostToken=%s,ingress.hosts={%s},ingress.annotations.nginx\\.ingress\\.kubernetes\\.io/whitelist-source-range=%s", kubecostToken, kubecostDNS, strings.Join(k.allowCIDRRangeList, "\\,"))

	return &helmDeployment{
		chartDeploymentName: "cost-analyzer",
		chartName:           "kubecost/cost-analyzer",
		namespace:           "kubecost",
		kubeconfigPath:      k.kubeconfigPath,
		setArgument:         helmValueArguments,
		logger:              k.logger,
		desiredVersion:      k.desiredVersion,
	}
}

func (k *kubecost) Name() string {
	return model.KubecostCanonicalName
}
