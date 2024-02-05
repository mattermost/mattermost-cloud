// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package utility

import (
	"context"
	"fmt"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/provisioner/prometheus"
	"github.com/mattermost/mattermost-cloud/internal/tools/argocd"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/git"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type unmanaged struct {
	awsClient          aws.AWS
	gitClient          git.Client
	argocdClient       argocd.Client
	kubeconfigPath     string
	allowCIDRRangeList []string
	cluster            *model.Cluster
	logger             log.FieldLogger
	utilityName        string
	tempDir            string
}

func newUnmanagedHandle(utilityName, kubeconfigPath, tempDir string, allowCIDRRangeList []string, cluster *model.Cluster, awsClient aws.AWS, gitClient git.Client, argocdClient argocd.Client, logger log.FieldLogger) *unmanaged {
	return &unmanaged{
		awsClient:          awsClient,
		gitClient:          gitClient,
		argocdClient:       argocdClient,
		kubeconfigPath:     kubeconfigPath,
		allowCIDRRangeList: allowCIDRRangeList,
		cluster:            cluster,
		tempDir:            tempDir,
		utilityName:        utilityName,
		logger: logger.WithFields(log.Fields{
			"cluster-utility": utilityName,
			"unmanaged":       true,
		}),
	}
}

func (u *unmanaged) ValuesPath() string {
	return ""
}

func (u *unmanaged) CreateOrUpgrade() error {
	u.logger.WithField("unmanaged-action", "create").Info("Utility is unmanaged; deploying with argocd...")

	privateDomainName, err := u.awsClient.GetPrivateZoneDomainName(u.logger)
	if err != nil {
		return errors.Wrap(err, "unable to lookup private zone name")
	}

	k8sClient, err := k8s.NewFromFile(u.kubeconfigPath, u.logger)
	if err != nil {
		return errors.Wrap(err, "failed to set up the k8s client")
	}

	switch u.Name() {
	case model.PgbouncerCanonicalName:
		err := deployManifests(k8sClient, u.logger)
		if err != nil {
			return err
		}
		if err := u.utiliyArgocdDeploy(u.Name()); err != nil {
			return errors.Wrapf(err, "failed to provision %s utility", u.Name())
		}

	case model.NginxCanonicalName, model.NginxInternalCanonicalName:
		if err := u.utiliyArgocdDeploy(u.Name()); err != nil {
			return errors.Wrapf(err, "failed to provision %s utility", u.Name())
		}

		endpoint, elbType, err := getElasticLoadBalancerInfo(u.Name(), u.logger, u.kubeconfigPath)
		if err != nil {
			return errors.Wrap(err, "couldn't get the loadbalancer endpoint (nginx-internal)")
		}

		if err := addLoadBalancerNameTag(u.awsClient.GetLoadBalancerAPIByType(elbType), endpoint); err != nil {
			return errors.Wrapf(err, "failed to add loadbalancer name tag (%s)", u.Name())
		}

	case model.PrometheusOperatorCanonicalName:
		logger := u.logger.WithField("prometheus-action", "create")

		_, err = k8sClient.CreateOrUpdateNamespace(prometheus.Namespace)
		if err != nil {
			return errors.Wrapf(err, "failed to create the prometheus namespace")
		}

		secretData := map[string]interface{}{
			"type": "s3",
			"config": map[string]interface{}{
				"bucket":       fmt.Sprintf("cloud-%s-prometheus-metrics", u.awsClient.GetCloudEnvironmentName()),
				"endpoint":     u.awsClient.GetS3RegionURL(),
				"aws_sdk_auth": true,
				"sse_config": map[string]string{
					"type": "SSE-S3",
				},
			},
		}

		secret, err := yaml.Marshal(secretData)
		if err != nil {
			return errors.Wrap(err, "thanos objstore secret yaml marshal failed")
		}

		thanosObjStoreSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name: "thanos-objstore-config",
			},
			StringData: map[string]string{
				"thanos.yaml": string(secret),
			},
		}

		_, err = k8sClient.CreateOrUpdateSecret(prometheus.Namespace, thanosObjStoreSecret)
		if err != nil {
			return errors.Wrapf(err, "failed to create the Thanos object storage secret")
		}

		if err := u.utiliyArgocdDeploy(u.Name()); err != nil {
			return errors.Wrapf(err, "failed to provision %s utility", u.Name())
		}

		app := "prometheus"
		dns := fmt.Sprintf("%s.%s.%s", u.cluster.ID, app, privateDomainName)

		if u.awsClient.IsProvisionedPrivateCNAME(dns, u.logger) {
			u.logger.Debugln("CNAME was already provisioned for prometheus")
			return nil
		}

		u.logger.Debugln("CNAME was not provisioned for prometheus")
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(180)*time.Second)
		defer cancel()

		endpoint, err := getPrivateLoadBalancerEndpoint(ctx, namespaceNginxInternal, logger.WithField("prometheus-action", "create"), u.kubeconfigPath)
		if err != nil {
			return errors.Wrap(err, "couldn't get the load balancer endpoint (nginx) for Prometheus")
		}

		logger.Infof("Registering DNS %s for %s", dns, app)
		err = u.awsClient.CreatePrivateCNAME(dns, []string{endpoint}, logger.WithField("prometheus-dns-create", dns))
		if err != nil {
			return errors.Wrap(err, "failed to create a CNAME to point to Prometheus")
		}

	case model.ThanosCanonicalName:
		logger := u.logger.WithField("thanos-action", "create")

		app := "thanos"
		dns := fmt.Sprintf("%s.%s.%s", u.cluster.ID, app, privateDomainName)
		grpcDNS := fmt.Sprintf("%s-grpc.%s.%s", u.cluster.ID, app, privateDomainName)

		if u.awsClient.IsProvisionedPrivateCNAME(dns, logger) {
			logger.Debugln("Main CNAME was already provisioned for thanos")
		} else {
			logger.Debugln("Main CNAME was not provisioned for thanos")
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(120)*time.Second)
			defer cancel()

			endpoint, err := getPrivateLoadBalancerEndpoint(ctx, namespaceNginxInternal, logger.WithField("thanos-action", "create"), u.kubeconfigPath)
			if err != nil {
				return errors.Wrap(err, "couldn't get the load balancer endpoint (nginx) for Thanos")
			}

			logger.Infof("Registering DNS %s for %s", dns, app)
			err = u.awsClient.CreatePrivateCNAME(dns, []string{endpoint}, logger.WithField("thanos-dns-create", dns))
			if err != nil {
				return errors.Wrap(err, "failed to create a CNAME to point to Thanos")
			}
		}

		if err := u.utiliyArgocdDeploy(u.Name()); err != nil {
			return errors.Wrapf(err, "failed to provision %s utility", u.Name())
		}

		if u.awsClient.IsProvisionedPrivateCNAME(grpcDNS, logger) {
			logger.Debugln("GRPC CNAME was already provisioned for thanos")
		} else {
			logger.Debugln("GRPC CNAME was not provisioned for thanos")
			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(120)*time.Second)
			defer cancel()

			endpoint, err := getPrivateLoadBalancerEndpoint(ctx, prometheus.Namespace, logger.WithField("thanos-action", "create"), u.kubeconfigPath)
			if err != nil {
				return errors.Wrap(err, "couldn't get the load balancer endpoint for Thanos")
			}

			logger.Infof("Registering GRPC DNS %s for %s", grpcDNS, app)
			err = u.awsClient.CreatePrivateCNAME(grpcDNS, []string{endpoint}, logger.WithField("thanos-dns-create", grpcDNS))
			if err != nil {
				return errors.Wrap(err, "failed to create a CNAME to point to Thanos GRPC")
			}
		}
	default:
		u.logger.WithFields(log.Fields{
			"unmanaged-action": "skip",
			"utility":          u.Name(),
		}).Info("Utility has already defined in argocd; skippping...")
	}

	return nil
}

func (u *unmanaged) Destroy() error {
	u.logger.WithField("unmanaged-action", "destroy").Info("Utility is unmanaged; skippping...")

	privateDomainName, err := u.awsClient.GetPrivateZoneDomainName(u.logger)
	if err != nil {
		return errors.Wrap(err, "unable to lookup private zone name")
	}

	switch u.Name() {
	case model.PrometheusOperatorCanonicalName:
		logger := u.logger.WithField("prometheus-action", "destroy")

		app := "prometheus"
		dns := fmt.Sprintf("%s.%s.%s", u.cluster.ID, app, privateDomainName)

		logger.Infof("Deleting Route53 DNS Record for %s", app)
		err = u.awsClient.DeletePrivateCNAME(dns, logger.WithField("prometheus-dns-delete", dns))
		if err != nil {
			return errors.Wrap(err, "failed to delete Route53 DNS record")
		}
	case model.ThanosCanonicalName:
		logger := u.logger.WithField("thanos-action", "destroy")

		app := "thanos"
		dns := fmt.Sprintf("%s.%s.%s", u.cluster.ID, app, privateDomainName)

		logger.WithField("dns", dns).Infof("Deleting Route53 DNS Record for %s", app)
		err = u.awsClient.DeletePrivateCNAME(dns, logger.WithField("thanos-dns-delete", dns))
		if err != nil {
			return errors.Wrap(err, "failed to delete Route53 DNS record")
		}

		grpcDNS := fmt.Sprintf("%s-grpc.%s.%s", u.cluster.ID, app, privateDomainName)
		logger.WithField("grpcDNS", grpcDNS).Infof("Deleting GRPC Route53 DNS Record for %s", app)
		err = u.awsClient.DeletePrivateCNAME(grpcDNS, logger.WithField("thanos-dns-delete", grpcDNS))
		if err != nil {
			return errors.Wrap(err, "failed to delete GRPC Route53 DNS record")
		}
	}

	return nil
}

func (u *unmanaged) Migrate() error {
	return nil
}

func (u *unmanaged) Name() string {
	return u.utilityName
}

func (u *unmanaged) DesiredVersion() *model.HelmUtilityVersion {
	return &model.HelmUtilityVersion{Chart: model.UnmanagedUtilityVersion}
}

func (u *unmanaged) ActualVersion() *model.HelmUtilityVersion {
	return &model.HelmUtilityVersion{Chart: model.UnmanagedUtilityVersion}
}

func (u *unmanaged) utiliyArgocdDeploy(utilityName string) error {
	if err := ProvisionUtilityArgocd(utilityName, u.tempDir, u.cluster.ID, u.allowCIDRRangeList, u.awsClient, u.gitClient, u.argocdClient, u.logger); err != nil {
		return errors.Wrapf(err, "failed to provision %s utility", utilityName)
	}
	return nil
}
