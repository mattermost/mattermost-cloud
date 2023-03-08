// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package provisioner

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/internal/tools/terraform"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	rotatorModel "github.com/mattermost/rotator/model"
	"github.com/mattermost/rotator/rotator"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// DefaultKubernetesVersion is the default value for a kubernetes cluster
// version value.
const (
	DefaultKubernetesVersion = "0.0.0"
	igFilename               = "ig-nodes.yaml"
)

// PrepareCluster ensures a cluster object is ready for provisioning.
func (provisioner *KopsProvisioner) PrepareCluster(cluster *model.Cluster) bool {
	// Don't regenerate the name if already set.
	if cluster.ProvisionerMetadataKops.Name != "" {
		return false
	}

	// Generate the kops name using the cluster id.
	cluster.ProvisionerMetadataKops.Name = fmt.Sprintf("%s-kops.k8s.local", cluster.ID)

	return true
}

// CreateCluster creates a cluster using kops and terraform.
func (provisioner *KopsProvisioner) CreateCluster(cluster *model.Cluster) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	kopsMetadata := cluster.ProvisionerMetadataKops

	err := kopsMetadata.ValidateChangeRequest()
	if err != nil {
		return errors.Wrap(err, "KopsMetadata ChangeRequest failed validation")
	}

	if kopsMetadata.ChangeRequest.AMI != "" && kopsMetadata.ChangeRequest.AMI != "latest" {
		var isAMIValid bool
		isAMIValid, err = provisioner.awsClient.IsValidAMI(kopsMetadata.ChangeRequest.AMI, logger)
		if err != nil {
			return errors.Wrapf(err, "error checking the AWS AMI image %s", kopsMetadata.ChangeRequest.AMI)
		}
		if !isAMIValid {
			return errors.Errorf("invalid AWS AMI image %s", kopsMetadata.ChangeRequest.AMI)
		}
	}

	cncVPCName := fmt.Sprintf("mattermost-cloud-%s-command-control", provisioner.awsClient.GetCloudEnvironmentName())
	cncVPCCIDR, err := provisioner.awsClient.GetCIDRByVPCTag(cncVPCName, logger)
	if err != nil {
		return errors.Wrapf(err, "failed to get the CIDR for the VPC Name %s", cncVPCName)
	}
	allowSSHCIDRS := []string{cncVPCCIDR}
	allowSSHCIDRS = append(allowSSHCIDRS, provisioner.params.VpnCIDRList...)

	logger.WithField("name", kopsMetadata.Name).Info("Creating cluster")
	kops, err := kops.New(provisioner.params.S3StateStore, logger)
	if err != nil {
		return err
	}
	defer kops.Close()

	var clusterResources aws.ClusterResources
	if kopsMetadata.ChangeRequest.VPC != "" && provisioner.params.UseExistingAWSResources {
		clusterResources, err = provisioner.awsClient.ClaimVPC(kopsMetadata.ChangeRequest.VPC, cluster, provisioner.params.Owner, logger)
		if err != nil {
			return errors.Wrap(err, "couldn't claim VPC")
		}
	} else if provisioner.params.UseExistingAWSResources {
		clusterResources, err = provisioner.awsClient.GetAndClaimVpcResources(cluster, provisioner.params.Owner, logger)
		if err != nil {
			return err
		}
	}

	err = kops.CreateCluster(
		kopsMetadata.Name,
		cluster.Provider,
		kopsMetadata.ChangeRequest,
		cluster.ProviderMetadataAWS.Zones,
		clusterResources.PrivateSubnetIDs,
		clusterResources.PublicSubnetsIDs,
		clusterResources.MasterSecurityGroupIDs,
		clusterResources.WorkerSecurityGroupIDs,
		allowSSHCIDRS,
	)
	// release VPC resources
	if err != nil {
		releaseErr := provisioner.awsClient.ReleaseVpc(cluster, logger)
		if releaseErr != nil {
			logger.WithError(releaseErr).Error("Unable to release VPC")
		}

		return errors.Wrap(err, "unable to create kops cluster")
	}
	terraformClient, err := terraform.New(kops.GetOutputDirectory(), provisioner.params.S3StateStore, logger)
	if err != nil {
		return err
	}
	defer terraformClient.Close()

	err = terraformClient.Init(kopsMetadata.Name)
	if err != nil {
		return err
	}

	err = terraformClient.ApplyTarget(fmt.Sprintf("aws_internet_gateway.%s-kops-k8s-local", cluster.ID))
	if err != nil {
		return err
	}

	err = terraformClient.ApplyTarget(fmt.Sprintf("aws_elb.api-%s-kops-k8s-local", cluster.ID))
	if err != nil {
		return err
	}

	// TODO: read from config file
	logger.Info("Updating kubelet options")

	setValue := "spec.kubelet.authenticationTokenWebhook=true"
	err = kops.SetCluster(kopsMetadata.Name, setValue)
	if err != nil {
		return errors.Wrapf(err, "failed to set %s", setValue)
	}
	setValue = "spec.kubelet.authorizationMode=Webhook"
	err = kops.SetCluster(kopsMetadata.Name, setValue)
	if err != nil {
		return errors.Wrapf(err, "failed to set %s", setValue)
	}
	if kopsMetadata.ChangeRequest.MaxPodsPerNode != 0 {
		logger.Infof("Updating max pods per node to %d", kopsMetadata.ChangeRequest.MaxPodsPerNode)
		setValue = fmt.Sprintf("spec.kubelet.maxPods=%d", kopsMetadata.ChangeRequest.MaxPodsPerNode)
		err = kops.SetCluster(kopsMetadata.Name, setValue)
		if err != nil {
			return errors.Wrapf(err, "failed to set %s", setValue)
		}
	}

	if kopsMetadata.ChangeRequest.Networking == "calico" {
		logger.Info("Updating calico options")
		setValue = "spec.networking.calico.prometheusMetricsEnabled=true"
		err = kops.SetCluster(kopsMetadata.Name, setValue)
		if err != nil {
			return errors.Wrapf(err, "failed to set %s", setValue)
		}
		setValue = "spec.networking.calico.prometheusMetricsPort=9091"
		err = kops.SetCluster(kopsMetadata.Name, setValue)
		if err != nil {
			return errors.Wrapf(err, "failed to set %s", setValue)
		}
		setValue = "spec.networking.calico.prometheusGoMetricsEnabled=true"
		err = kops.SetCluster(kopsMetadata.Name, setValue)
		if err != nil {
			return errors.Wrapf(err, "failed to set %s", setValue)
		}
		setValue = "spec.networking.calico.prometheusProcessMetricsEnabled=true"
		err = kops.SetCluster(kopsMetadata.Name, setValue)
		if err != nil {
			return errors.Wrapf(err, "failed to set %s", setValue)
		}
		setValue = "spec.networking.calico.typhaPrometheusMetricsEnabled=true"
		err = kops.SetCluster(kopsMetadata.Name, setValue)
		if err != nil {
			return errors.Wrapf(err, "failed to set %s", setValue)
		}
		setValue = "spec.networking.calico.typhaPrometheusMetricsPort=9093"
		err = kops.SetCluster(kopsMetadata.Name, setValue)
		if err != nil {
			return errors.Wrapf(err, "failed to set %s", setValue)
		}
		setValue = "spec.networking.calico.typhaReplicas=2"
		err = kops.SetCluster(kopsMetadata.Name, setValue)
		if err != nil {
			return errors.Wrapf(err, "failed to set %s", setValue)
		}
	}

	if len(provisioner.params.EtcdManagerEnv) > 0 {
		var override []string
		var overrideIndex int
		for key, val := range provisioner.params.EtcdManagerEnv {
			override = append(override, "spec.etcdClusters[0].manager.env=")
			envName := fmt.Sprintf("spec.etcdClusters[0].manager.env[%d].name=%s", overrideIndex, key)
			envValue := fmt.Sprintf("spec.etcdClusters[0].manager.env[%d].value=%s", overrideIndex, val)
			override = append(override, envName, envValue)
			overrideIndex++
		}

		logger.Infof("Adding environment variables to etcd cluster manager")
		err = kops.SetCluster(kopsMetadata.Name, strings.Join(override, ","))
		if err != nil {
			return errors.Wrap(err, "failed to set etcd environment variables")
		}
	}

	err = updateKopsInstanceGroupValue(kops, kopsMetadata, "spec.instanceMetadata.httpTokens=optional")
	if err != nil {
		return errors.Wrap(err, "failed to update kops instance group instance Metadata")
	}

	err = kops.UpdateCluster(kopsMetadata.Name, kops.GetOutputDirectory())
	if err != nil {
		return err
	}

	err = provisioner.awsClient.FixSubnetTagsForVPC(clusterResources.VpcID, logger)
	if err != nil {
		return err
	}

	err = terraformClient.Apply()
	if err != nil {
		return err
	}

	// TODO: Rework this as we make the API calls asynchronous.
	wait := 1000
	logger.Infof("Waiting up to %d seconds for k8s cluster to become ready...", wait)
	err = kops.WaitForKubernetesReadiness(kopsMetadata.Name, wait)
	if err != nil {
		// Run non-silent validate one more time to log final cluster state
		// and return original timeout error.
		kops.ValidateCluster(kopsMetadata.Name, false)
		return err
	}

	logger.WithField("name", kopsMetadata.Name).Info("Successfully deployed kubernetes")

	iamRole := fmt.Sprintf("nodes.%s", kopsMetadata.Name)
	err = provisioner.awsClient.AttachPolicyToRole(iamRole, aws.CustomNodePolicyName, logger)
	if err != nil {
		return errors.Wrap(err, "unable to attach custom node policy")
	}

	err = provisioner.awsClient.AttachPolicyToRole(iamRole, aws.VeleroNodePolicyName, logger)
	if err != nil {
		return errors.Wrap(err, "unable to attach velero node policy")
	}

	iamRole = fmt.Sprintf("masters.%s", kopsMetadata.Name)
	err = provisioner.awsClient.AttachPolicyToRole(iamRole, aws.CustomNodePolicyName, logger)
	if err != nil {
		return errors.Wrap(err, "unable to attach custom node policy to master")
	}

	ugh, err := newUtilityGroupHandle(provisioner.params, kops.GetKubeConfigPath(), cluster, provisioner.awsClient, logger)
	if err != nil {
		return err
	}

	return ugh.CreateUtilityGroup()
}

// CheckClusterCreated is a noop for KopsProvisioner.
func (provisioner *KopsProvisioner) CheckClusterCreated(cluster *model.Cluster) (bool, error) {
	// TODO: this is currently not implemented for kops.
	// Entire waiting logic happens as part of cluster creation therefore we
	// just skip this step and report cluster as created.
	return true, nil
}

// CheckNodesCreated is a noop for KopsProvisioner.
func (provisioner *KopsProvisioner) CheckNodesCreated(cluster *model.Cluster) (bool, error) {
	// TODO: this is currently not implemented for kops.
	// Entire waiting logic happens as part of cluster creation therefore we
	// just skip this step and report cluster as created.
	return true, nil
}

// ProvisionCluster installs all the baseline kubernetes resources needed for
// managing installations. This can be called on an already-provisioned cluster
// to re-provision with the newest version of the resources.
func (provisioner *KopsProvisioner) ProvisionCluster(cluster *model.Cluster) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	logger.Info("Provisioning cluster")
	kopsClient, err := provisioner.getCachedKopsClient(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get kops client from cache")
	}
	defer provisioner.invalidateCachedKopsClientOnError(err, cluster.ProvisionerMetadataKops.Name, logger)

	return provisionCluster(cluster, kopsClient.GetKubeConfigPath(), provisioner.awsClient, provisioner.params, provisioner.store, logger)
}

// UpgradeCluster upgrades a cluster to the latest recommended production ready k8s version.
func (provisioner *KopsProvisioner) UpgradeCluster(cluster *model.Cluster) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	kopsMetadata := cluster.ProvisionerMetadataKops

	err := kopsMetadata.ValidateChangeRequest()
	if err != nil {
		return errors.Wrap(err, "KopsMetadata ChangeRequest failed validation")
	}

	if kopsMetadata.ChangeRequest.AMI != "" && kopsMetadata.ChangeRequest.AMI != "latest" {
		var isAMIValid bool
		isAMIValid, err = provisioner.awsClient.IsValidAMI(kopsMetadata.ChangeRequest.AMI, logger)
		if err != nil {
			return errors.Wrapf(err, "error checking the AWS AMI image %s", kopsMetadata.ChangeRequest.AMI)
		}
		if !isAMIValid {
			return errors.Errorf("invalid AWS AMI image %s", kopsMetadata.ChangeRequest.AMI)
		}
	}

	kops, err := kops.New(provisioner.params.S3StateStore, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	switch kopsMetadata.ChangeRequest.Version {
	case "":
		logger.Info("Skipping kubernetes cluster version update")
	case "latest":
		logger.Info("Updating kubernetes to latest stable version")
		err = kops.UpgradeCluster(kopsMetadata.Name)
		if err != nil {
			return err
		}
	default:
		logger.Infof("Updating kubernetes to version %s", kopsMetadata.ChangeRequest.Version)
		setValue := fmt.Sprintf("spec.kubernetesVersion=%s", kopsMetadata.ChangeRequest.Version)
		err = kops.SetCluster(kopsMetadata.Name, setValue)
		if err != nil {
			return err
		}
	}

	err = updateKopsInstanceGroupAMIs(kops, kopsMetadata, logger)
	if err != nil {
		return errors.Wrap(err, "failed to update kops instance group AMIs")
	}

	// TODO: read from config file
	// TODO: check if those configs are already or remove this when we update all clusters
	logger.Info("Updating kubelet options")

	setValue := "spec.kubelet.authenticationTokenWebhook=true"
	err = kops.SetCluster(kopsMetadata.Name, setValue)
	if err != nil {
		return errors.Wrapf(err, "failed to set %s", setValue)
	}
	setValue = "spec.kubelet.authorizationMode=Webhook"
	err = kops.SetCluster(kopsMetadata.Name, setValue)
	if err != nil {
		return errors.Wrapf(err, "failed to set %s", setValue)
	}
	if kopsMetadata.ChangeRequest.MaxPodsPerNode != 0 {
		logger.Infof("Updating max pods per node to %d", kopsMetadata.ChangeRequest.MaxPodsPerNode)
		setValue = fmt.Sprintf("spec.kubelet.maxPods=%d", kopsMetadata.ChangeRequest.MaxPodsPerNode)
		err = kops.SetCluster(kopsMetadata.Name, setValue)
		if err != nil {
			return errors.Wrapf(err, "failed to set %s", setValue)
		}
	}

	err = kops.UpdateCluster(kopsMetadata.Name, kops.GetOutputDirectory())
	if err != nil {
		return err
	}

	err = provisioner.awsClient.FixSubnetTagsForVPC(kopsMetadata.VPC, logger)
	if err != nil {
		return err
	}

	terraformClient, err := terraform.New(kops.GetOutputDirectory(), provisioner.params.S3StateStore, logger)
	if err != nil {
		return err
	}
	defer terraformClient.Close()

	err = terraformClient.Init(kopsMetadata.Name)
	if err != nil {
		return err
	}

	err = verifyTerraformAndKopsMatch(kopsMetadata.Name, terraformClient, logger)
	if err != nil {
		return err
	}

	logger.Info("Upgrading cluster")

	err = terraformClient.Plan()
	if err != nil {
		return err
	}
	err = terraformClient.Apply()
	if err != nil {
		return err
	}

	if cluster.ProvisionerMetadataKops.RotatorRequest.Config != nil {
		if *cluster.ProvisionerMetadataKops.RotatorRequest.Config.UseRotator {
			logger.Info("Using node rotator for node upgrade")
			err = provisioner.RotateClusterNodes(cluster)
			if err != nil {
				return err
			}
		}
	}

	err = kops.RollingUpdateCluster(kopsMetadata.Name)
	if err != nil {
		return err
	}

	// TODO: Rework this as we make the API calls asynchronous.
	wait := 1000
	if wait > 0 {
		logger.Infof("Waiting up to %d seconds for k8s cluster to become ready...", wait)
		err = kops.WaitForKubernetesReadiness(kopsMetadata.Name, wait)
		if err != nil {
			// Run non-silent validate one more time to log final cluster state
			// and return original timeout error.
			kops.ValidateCluster(kopsMetadata.Name, false)
			return err
		}
	}

	iamRole := fmt.Sprintf("nodes.%s", kopsMetadata.Name)
	err = provisioner.awsClient.AttachPolicyToRole(iamRole, aws.CustomNodePolicyName, logger)
	if err != nil {
		return errors.Wrap(err, "unable to attach custom node policy")
	}

	err = provisioner.awsClient.AttachPolicyToRole(iamRole, aws.VeleroNodePolicyName, logger)
	if err != nil {
		return errors.Wrap(err, "unable to attach velero node policy")
	}

	logger.Info("Successfully upgraded cluster")

	return nil
}

// RotateClusterNodes rotates k8s cluster nodes using the Mattermost node rotator
func (provisioner *KopsProvisioner) RotateClusterNodes(cluster *model.Cluster) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	kopsClient, err := provisioner.getCachedKopsClient(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get kops client from cache")
	}
	defer provisioner.invalidateCachedKopsClientOnError(err, cluster.ProvisionerMetadataKops.Name, logger)

	k8sClient, err := k8s.NewFromFile(kopsClient.GetKubeConfigPath(), logger)
	if err != nil {
		return err
	}
	clientset, err := kubernetes.NewForConfig(k8sClient.GetConfig())
	if err != nil {
		return err
	}

	clusterRotator := rotatorModel.Cluster{
		ClusterID:               cluster.ID,
		MaxScaling:              *cluster.ProvisionerMetadataKops.RotatorRequest.Config.MaxScaling,
		RotateMasters:           true,
		RotateWorkers:           true,
		MaxDrainRetries:         *cluster.ProvisionerMetadataKops.RotatorRequest.Config.MaxDrainRetries,
		EvictGracePeriod:        *cluster.ProvisionerMetadataKops.RotatorRequest.Config.EvictGracePeriod,
		WaitBetweenRotations:    *cluster.ProvisionerMetadataKops.RotatorRequest.Config.WaitBetweenRotations,
		WaitBetweenDrains:       *cluster.ProvisionerMetadataKops.RotatorRequest.Config.WaitBetweenDrains,
		WaitBetweenPodEvictions: *cluster.ProvisionerMetadataKops.RotatorRequest.Config.WaitBetweenPodEvictions,
		ClientSet:               clientset,
	}

	rotatorMetadata := cluster.ProvisionerMetadataKops.RotatorRequest.Status
	if rotatorMetadata == nil {
		rotatorMetadata = &rotator.RotatorMetadata{}
	}
	rotatorMetadata, err = rotator.InitRotateCluster(&clusterRotator, rotatorMetadata, logger)
	if err != nil {
		cluster.ProvisionerMetadataKops.RotatorRequest.Status = rotatorMetadata
		return err
	}

	return nil
}

// ResizeCluster resizes a cluster.
func (provisioner *KopsProvisioner) ResizeCluster(cluster *model.Cluster) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	kopsMetadata := cluster.ProvisionerMetadataKops

	err := kopsMetadata.ValidateChangeRequest()
	if err != nil {
		return errors.Wrap(err, "KopsMetadata ChangeRequest failed validation")
	}

	kops, err := kops.New(provisioner.params.S3StateStore, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create kops wrapper")
	}
	defer kops.Close()

	err = kops.UpdateCluster(kopsMetadata.Name, kops.GetOutputDirectory())
	if err != nil {
		return err
	}

	err = provisioner.awsClient.FixSubnetTagsForVPC(kopsMetadata.VPC, logger)
	if err != nil {
		return err
	}

	terraformClient, err := terraform.New(kops.GetOutputDirectory(), provisioner.params.S3StateStore, logger)
	if err != nil {
		return err
	}
	defer terraformClient.Close()

	err = terraformClient.Init(kopsMetadata.Name)
	if err != nil {
		return err
	}

	err = verifyTerraformAndKopsMatch(kopsMetadata.Name, terraformClient, logger)
	if err != nil {
		return err
	}

	logger.Info("Resizing cluster")

	for igName, changeMetadata := range kopsMetadata.GetWorkerNodesResizeChanges() {
		kopsSetActions := kopsMetadata.GetKopsResizeSetActionsFromChanges(changeMetadata, igName)
		for _, action := range kopsSetActions {
			logger.Debugf("Updating instance group %s with kops set %s", igName, action)
			err = kops.SetInstanceGroup(kopsMetadata.Name, igName, action)
			if err != nil {
				return errors.Wrapf(err, "failed to update instance group with %s", action)
			}
		}
	}

	err = kops.UpdateCluster(kopsMetadata.Name, kops.GetOutputDirectory())
	if err != nil {
		return err
	}

	err = provisioner.awsClient.FixSubnetTagsForVPC(kopsMetadata.VPC, logger)
	if err != nil {
		return err
	}

	err = terraformClient.Plan()
	if err != nil {
		return err
	}
	err = terraformClient.Apply()
	if err != nil {
		return err
	}

	requiresClusterRotation, err := kops.RollingUpdateClusterRequired(kopsMetadata.Name)
	if err != nil {
		return err
	}

	if requiresClusterRotation {
		logger.Info("Rolling update is required")
		if cluster.ProvisionerMetadataKops.RotatorRequest.Config != nil {
			if *cluster.ProvisionerMetadataKops.RotatorRequest.Config.UseRotator {
				logger.Info("Using node rotator for node resize")
				err = provisioner.RotateClusterNodes(cluster)
				if err != nil {
					return err
				}
			}
		}
	}

	err = kops.RollingUpdateCluster(kopsMetadata.Name)
	if err != nil {
		return err
	}

	// TODO: Rework this as we make the API calls asynchronous.
	wait := 1000
	if wait > 0 {
		logger.Infof("Waiting up to %d seconds for k8s cluster to become ready...", wait)
		err = kops.WaitForKubernetesReadiness(kopsMetadata.Name, wait)
		if err != nil {
			// Run non-silent validate one more time to log final cluster state
			// and return original timeout error.
			kops.ValidateCluster(kopsMetadata.Name, false)
			return err
		}
	}

	iamRole := fmt.Sprintf("nodes.%s", kopsMetadata.Name)
	err = provisioner.awsClient.AttachPolicyToRole(iamRole, aws.CustomNodePolicyName, logger)
	if err != nil {
		return errors.Wrap(err, "unable to attach custom node policy")
	}

	err = provisioner.awsClient.AttachPolicyToRole(iamRole, aws.VeleroNodePolicyName, logger)
	if err != nil {
		return errors.Wrap(err, "unable to attach velero node policy")
	}

	logger.Info("Successfully resized cluster")

	return nil
}

// DeleteCluster deletes a previously created cluster using kops and terraform.
func (provisioner *KopsProvisioner) DeleteCluster(cluster *model.Cluster) (bool, error) {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	kopsMetadata := cluster.ProvisionerMetadataKops

	logger.Info("Deleting cluster")

	exists, err := provisioner.kopsClusterExists(kopsMetadata.Name, logger)
	if err != nil {
		return false, errors.Wrap(err, "failed to check if kops cluster exists")
	}
	if exists {
		err = provisioner.cleanupKopsCluster(cluster, logger)
		if err != nil {
			return false, errors.Wrap(err, "failed to delete kops cluster")
		}
	} else {
		logger.Infof("Kops cluster %s does not exist, assuming already deleted", kopsMetadata.Name)
	}

	err = provisioner.awsClient.ReleaseVpc(cluster, logger)
	if err != nil {
		return false, errors.Wrap(err, "failed to release cluster VPC")
	}

	provisioner.invalidateCachedKopsClient(kopsMetadata.Name, logger)

	logger.Info("Successfully deleted Kops cluster")

	return true, nil
}

// cleanupKopsCluster cleans up Kops cluster. Make sure cluster exists before calling this method.
func (provisioner *KopsProvisioner) cleanupKopsCluster(cluster *model.Cluster, logger logrus.FieldLogger) error {
	kopsMetadata := cluster.ProvisionerMetadataKops

	kopsClient, err := provisioner.getCachedKopsClient(kopsMetadata.Name, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get kops client from cache")
	}
	defer provisioner.invalidateCachedKopsClientOnError(err, kopsMetadata.Name, logger)

	ugh, err := newUtilityGroupHandle(provisioner.params, kopsClient.GetKubeConfigPath(), cluster, provisioner.awsClient, logger)
	if err != nil {
		return errors.Wrap(err, "couldn't create new utility group handle while deleting the cluster")
	}

	err = ugh.DestroyUtilityGroup()
	if err != nil {
		return errors.Wrap(err, "failed to destroy all services in the utility group")
	}

	iamRole := fmt.Sprintf("nodes.%s", kopsMetadata.Name)
	err = provisioner.awsClient.DetachPolicyFromRole(iamRole, aws.CustomNodePolicyName, logger)
	if err != nil {
		return errors.Wrap(err, "unable to detach custom node policy")
	}
	err = provisioner.awsClient.DetachPolicyFromRole(iamRole, aws.VeleroNodePolicyName, logger)
	if err != nil {
		return errors.Wrap(err, "unable to detach velero node policy")
	}

	_, err = kopsClient.GetCluster(kopsMetadata.Name)
	if err != nil {
		return errors.Wrap(err, "failed to get kops cluster for deletion")
	}

	err = kopsClient.UpdateCluster(kopsMetadata.Name, kopsClient.GetOutputDirectory())
	if err != nil {
		return errors.Wrap(err, "failed to run kops update")
	}

	err = provisioner.awsClient.FixSubnetTagsForVPC(kopsMetadata.VPC, logger)
	if err != nil {
		return err
	}

	terraformClient, err := terraform.New(kopsClient.GetOutputDirectory(), provisioner.params.S3StateStore, logger)
	if err != nil {
		return errors.Wrap(err, "failed to create terraform wrapper")
	}
	defer terraformClient.Close()

	err = terraformClient.Init(kopsMetadata.Name)
	if err != nil {
		return errors.Wrap(err, "failed to init terraform")
	}

	err = verifyTerraformAndKopsMatch(kopsMetadata.Name, terraformClient, logger)
	if err != nil {
		logger.WithError(err).Error("Proceeding with cluster deletion despite failing terraform output match check")
	}

	err = terraformClient.Destroy()
	if err != nil {
		return errors.Wrap(err, "failed to run terraform destroy")
	}

	err = kopsClient.DeleteCluster(kopsMetadata.Name)
	if err != nil {
		return errors.Wrap(err, "failed to run kops delete")
	}

	logger.Infof("Kops cluster %s deleted", kopsMetadata.Name)

	return nil
}

// GetClusterResources returns a snapshot of resources of a given cluster.
func (provisioner Provisioner) GetClusterResources(cluster *model.Cluster, onlySchedulable bool, logger logrus.FieldLogger) (*k8s.ClusterResources, error) {
	logger = logger.WithField("cluster", cluster.ID)

	configLocation, err := provisioner.getClusterKubecfg(cluster)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get kube config path")
	}
	return getClusterResources(configLocation, onlySchedulable, logger)
}

// RefreshKopsMetadata updates the kops metadata of a cluster with the current
// values of the running cluster.
func (provisioner *KopsProvisioner) RefreshKopsMetadata(cluster *model.Cluster) error {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	logger.Info("Refreshing kops metadata")

	kopsClient, err := provisioner.getCachedKopsClient(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return errors.Wrap(err, "failed to get kops client from cache")
	}
	defer provisioner.invalidateCachedKopsClientOnError(err, cluster.ProvisionerMetadataKops.Name, logger)

	k8sClient, err := k8s.NewFromFile(kopsClient.GetKubeConfigPath(), logger)
	if err != nil {
		return errors.Wrap(err, "failed to construct k8s client")
	}

	versionInfo, err := k8sClient.Clientset.Discovery().ServerVersion()
	if err != nil {
		return errors.Wrap(err, "failed to get kubernetes version")
	}

	// The GitVersion string usually looks like v1.14.2 so we trim the "v" off
	// to match the version syntax used in kops.
	cluster.ProvisionerMetadataKops.Version = strings.TrimLeft(versionInfo.GitVersion, "v")

	err = kopsClient.UpdateMetadata(cluster.ProvisionerMetadataKops)
	if err != nil {
		return errors.Wrap(err, "failed to update metadata from kops state")
	}

	return nil
}

func (provisioner *KopsProvisioner) getKubeConfigPath(cluster *model.Cluster) (string, error) {
	logger := provisioner.logger.WithField("cluster", cluster.ID)

	configLocation, err := provisioner.getCachedKopsClusterKubecfg(cluster.ProvisionerMetadataKops.Name, logger)
	if err != nil {
		return "", errors.Wrap(err, "failed to get kops config from cache")
	}

	return configLocation, nil
}

func (provisioner *KopsProvisioner) getKubeClient(cluster *model.Cluster) (*k8s.KubeClient, error) {
	k8sClient, err := provisioner.k8sClient(cluster.ProvisionerMetadataKops.Name, provisioner.logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create k8s client")
	}

	return k8sClient, nil
}

// prepareSloth prepares sloth resources after prometheus utility is installed.
func prepareSloth(k8sClient *k8s.KubeClient, logger logrus.FieldLogger) error {
	files := []k8s.ManifestFile{
		{
			Path:            "manifests/sloth/crd_sloth.slok.dev_prometheusservicelevels.yaml",
			DeployNamespace: prometheusNamespace,
		},
		{
			Path:            "manifests/sloth/sloth.yaml",
			DeployNamespace: prometheusNamespace,
		},
	}

	err := k8sClient.CreateFromFiles(files)
	if err != nil {
		return errors.Wrapf(err, "failed to create sloth resources.")
	}
	wait := 240
	pods, err := k8sClient.GetPodsFromDeployment(prometheusNamespace, "sloth")
	if err != nil {
		return err
	}
	if len(pods.Items) == 0 {
		return fmt.Errorf("no pods found from sloth deployment")
	}

	for _, pod := range pods.Items {
		logger.Infof("Waiting up to %d seconds for %q pod %q to start...", wait, "sloth", pod.GetName())
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(wait)*time.Second)
		defer cancel()
		_, err := k8sClient.WaitForPodRunning(ctx, prometheusNamespace, pod.GetName())
		if err != nil {
			return err
		}
		logger.Infof("Successfully deployed service pod %q", pod.GetName())
	}

	return nil
}
