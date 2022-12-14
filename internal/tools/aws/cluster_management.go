// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// ClusterResources is a collection of AWS resources that will be used to create
// a kops cluster.
type ClusterResources struct {
	VpcID                  string
	VpcCIDR                string
	PrivateSubnetIDs       []string
	PublicSubnetsIDs       []string
	MasterSecurityGroupIDs []string
	WorkerSecurityGroupIDs []string
	CallsSecurityGroupIDs  []string
}

// IsValid returns whether or not ClusterResources is valid or not.
func (cr *ClusterResources) IsValid() error {
	if cr.VpcID == "" {
		return errors.New("vpc ID is empty")
	}
	if len(cr.PrivateSubnetIDs) == 0 {
		return errors.New("private subnet list is empty")
	}
	if len(cr.PublicSubnetsIDs) == 0 {
		return errors.New("public subnet list is empty")
	}
	if len(cr.MasterSecurityGroupIDs) == 0 {
		return errors.New("master security group list is empty")
	}
	if len(cr.WorkerSecurityGroupIDs) == 0 {
		return errors.New("worker security group list is empty")
	}
	if len(cr.CallsSecurityGroupIDs) == 0 {
		return errors.New("calls security group list is empty")
	}

	return nil
}

func (a *Client) getClusterResourcesForVPC(vpcID, vpcCIDR string, logger log.FieldLogger) (ClusterResources, error) {
	clusterResources := ClusterResources{
		VpcID:   vpcID,
		VpcCIDR: vpcCIDR,
	}

	baseFilter := []ec2Types.Filter{
		{
			Name:   aws.String("vpc-id"),
			Values: []string{vpcID},
		},
	}

	privateSubnetFilter := append(baseFilter, ec2Types.Filter{
		Name:   aws.String("tag:SubnetType"),
		Values: []string{"private"},
	})

	privateSubnets, err := a.GetSubnetsWithFilters(privateSubnetFilter)
	if err != nil {
		return clusterResources, err
	}

	for _, subnet := range privateSubnets {
		clusterResources.PrivateSubnetIDs = append(clusterResources.PrivateSubnetIDs, *subnet.SubnetId)
	}

	publicSubnetFilter := append(baseFilter, ec2Types.Filter{
		Name:   aws.String("tag:SubnetType"),
		Values: []string{"public"},
	})

	publicSubnets, err := a.GetSubnetsWithFilters(publicSubnetFilter)
	if err != nil {
		return clusterResources, err
	}

	for _, subnet := range publicSubnets {
		clusterResources.PublicSubnetsIDs = append(clusterResources.PublicSubnetsIDs, *subnet.SubnetId)
	}

	masterSGFilter := append(baseFilter, ec2Types.Filter{
		Name:   aws.String("tag:NodeType"),
		Values: []string{"master"},
	})

	masterSecurityGroups, err := a.GetSecurityGroupsWithFilters(masterSGFilter)
	if err != nil {
		return clusterResources, err
	}

	for _, securityGroup := range masterSecurityGroups {
		clusterResources.MasterSecurityGroupIDs = append(clusterResources.MasterSecurityGroupIDs, *securityGroup.GroupId)
	}

	workerSGFilter := append(baseFilter, ec2Types.Filter{
		Name:   aws.String("tag:NodeType"),
		Values: []string{"worker"},
	})

	workerSecurityGroups, err := a.GetSecurityGroupsWithFilters(workerSGFilter)
	if err != nil {
		return clusterResources, err
	}

	for _, securityGroup := range workerSecurityGroups {
		clusterResources.WorkerSecurityGroupIDs = append(clusterResources.WorkerSecurityGroupIDs, *securityGroup.GroupId)
	}

	callsSGFilter := append(baseFilter, ec2Types.Filter{
		Name:   aws.String("tag:NodeType"),
		Values: []string{"calls"},
	})

	callsSecurityGroups, err := a.GetSecurityGroupsWithFilters(callsSGFilter)
	if err != nil {
		return clusterResources, err
	}

	for _, securityGroup := range callsSecurityGroups {
		clusterResources.CallsSecurityGroupIDs = append(clusterResources.CallsSecurityGroupIDs, *securityGroup.GroupId)
	}

	err = clusterResources.IsValid()
	if err != nil {
		return clusterResources, errors.Wrapf(err, "VPC %s is misconfigured", clusterResources.VpcID)
	}

	return clusterResources, nil
}

// ClaimVPC claims specified VPC for specified cluster.
func (a *Client) ClaimVPC(vpcID string, cluster *model.Cluster, owner string, logger log.FieldLogger) (ClusterResources, error) {
	ctx := context.TODO()
	vpcOut, err := a.Service().ec2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{VpcIds: []string{vpcID}})
	if err != nil {
		return ClusterResources{}, errors.Wrap(err, "failed to describe vpc")
	}

	if len(vpcOut.Vpcs) == 0 {
		return ClusterResources{}, fmt.Errorf("couldn't find vpcs")
	}

	clusterResources, err := a.getClusterResourcesForVPC(vpcID, *vpcOut.Vpcs[0].CidrBlock, logger)
	if err != nil {
		return ClusterResources{}, errors.Wrap(err, "failed to get cluster resources for VPC")
	}

	err = a.claimVpc(clusterResources, cluster, owner, logger)
	if err != nil {
		return ClusterResources{}, errors.Wrap(err, "failed to claim VPC")
	}

	return clusterResources, nil
}

// GetAndClaimVpcResources creates ClusterResources from an available VPC and
// tags them appropriately.
func (a *Client) GetAndClaimVpcResources(cluster *model.Cluster, owner string, logger log.FieldLogger) (ClusterResources, error) {
	// First, check if a VPC has been claimed by this cluster. If only one has
	// already been claimed, then return that with no error.
	clusterAlreadyClaimedFilter := []ec2Types.Filter{
		{
			Name: aws.String(VpcAvailableTagKey),
			Values: []string{
				VpcAvailableTagValueFalse,
			},
		},
		{
			Name: aws.String(VpcClusterIDTagKey),
			Values: []string{
				cluster.ID,
			},
		},
	}
	clusterAlreadyClaimedVpcs, err := a.GetVpcsWithFilters(clusterAlreadyClaimedFilter)
	if err != nil {
		return ClusterResources{}, err
	}
	if len(clusterAlreadyClaimedVpcs) > 1 {
		return ClusterResources{}, fmt.Errorf("multiple VPCs (%d) have been claimed by cluster %s; aborting claim process", len(clusterAlreadyClaimedVpcs), cluster.ID)
	}
	if len(clusterAlreadyClaimedVpcs) == 1 {
		return a.getClusterResourcesForVPC(*clusterAlreadyClaimedVpcs[0].VpcId, *clusterAlreadyClaimedVpcs[0].CidrBlock, logger)
	}

	// This cluster has not already claimed a VPC. Continue with claiming process.
	totalVpcsFilter := []ec2Types.Filter{
		{
			Name: aws.String(VpcAvailableTagKey),
			Values: []string{
				VpcAvailableTagValueTrue,
				VpcAvailableTagValueFalse,
			},
		},
	}
	totalVpcs, err := a.GetVpcsWithFilters(totalVpcsFilter)
	if err != nil {
		return ClusterResources{}, err
	}
	totalVpcCount := len(totalVpcs)

	vpcFilters := []ec2Types.Filter{
		{
			Name:   aws.String(VpcAvailableTagKey),
			Values: []string{VpcAvailableTagValueTrue},
		},
	}

	vpcs, err := a.GetVpcsWithFilters(vpcFilters)
	if err != nil {
		return ClusterResources{}, err
	}
	availableVpcCount := len(vpcs)

	logger.Debugf("Claiming VPC: %d total, %d available", totalVpcCount, availableVpcCount)

	// Loop through the VPCs. Based on the filter above these should all be
	// valid so we will claim the first one. Before doing that a sanity check of
	// the VPCs resources will occur.
	for _, vpc := range vpcs {
		clusterResources, err := a.getClusterResourcesForVPC(*vpc.VpcId, *vpc.CidrBlock, logger)
		if err != nil {
			logger.Warn(err)
			continue
		}

		err = a.claimVpc(clusterResources, cluster, owner, logger)
		if err != nil {
			return clusterResources, err
		}

		return clusterResources, nil
	}

	return ClusterResources{}, fmt.Errorf("%d VPCs were returned as currently available; none of them were configured correctly", len(vpcs))
}

// GetVpcResources retrieves the VPC information for a particulary cluster.
func (a *Client) GetVpcResources(clusterID string, logger log.FieldLogger) (ClusterResources, error) {
	vpc, err := getVPCForCluster(clusterID, a)
	if err != nil {
		return ClusterResources{}, errors.Wrap(err, "failed to find cluster VPC")
	}

	return a.getClusterResourcesForVPC(*vpc.VpcId, *vpc.CidrBlock, logger)
}

// ReleaseVpc changes the tags on a VPC to mark it as "available" again.
func (a *Client) ReleaseVpc(cluster *model.Cluster, logger log.FieldLogger) error {
	return a.releaseVpc(cluster, logger)
}

// claimVpc will claim the given VPC for a cluster if a final race-check passes.
// The final race check does the following:
// - Requires the VPC to exist. #mindblown
// - VPC availabiltiy tag must be "true"
// - VPC cluster ID tag must by "none"
// If that conditions are not met, we will try to set this cluster as secondary in the VPC only if
// the `CloudSecondaryClusterID` is set to `none`.
func (a *Client) claimVpc(clusterResources ClusterResources, cluster *model.Cluster, owner string, logger log.FieldLogger) error {
	vpcFilter := []ec2Types.Filter{
		{
			Name:   aws.String("vpc-id"),
			Values: []string{clusterResources.VpcID},
		},
		{
			Name:   aws.String(VpcAvailableTagKey),
			Values: []string{VpcAvailableTagValueTrue},
		},
		{
			Name:   aws.String(VpcClusterIDTagKey),
			Values: []string{VpcClusterIDTagValueNone},
		},
	}
	vpcs, err := a.GetVpcsWithFilters(vpcFilter)
	if err != nil {
		return err
	}

	numVPCs := len(vpcs)
	var claimSecondaryCluster bool
	if numVPCs > 1 {
		return fmt.Errorf("query for VPC %s somehow returned multiple results", clusterResources.VpcID)
	}
	if numVPCs == 0 {
		vpcFilter = []ec2Types.Filter{
			{
				Name:   aws.String("vpc-id"),
				Values: []string{clusterResources.VpcID},
			},
			{
				Name:   aws.String(VpcAvailableTagKey),
				Values: []string{VpcAvailableTagValueFalse},
			},
			{
				Name:   aws.String(VpcSecondaryClusterIDTagKey),
				Values: []string{VpcClusterIDTagValueNone},
			},
		}

		vpcs, err = a.GetVpcsWithFilters(vpcFilter)
		if err != nil {
			return err
		}

		if len(vpcs) > 1 {
			return fmt.Errorf("query for secondary VPC %s somehow returned multiple results", clusterResources.VpcID)
		}

		if len(vpcs) == 1 {
			claimSecondaryCluster = true
		} else {
			return fmt.Errorf("couldn't claim VPC %s as primary nor secondary cluster", clusterResources.VpcID)
		}
	}

	if claimSecondaryCluster {
		err = a.TagResource(clusterResources.VpcID, trimTagPrefix(VpcSecondaryClusterIDTagKey), cluster.ID, logger)
		if err != nil {
			return errors.Wrapf(err, "unable to update %s", VpcClusterIDTagKey)
		}
		// TODO: what about ownership when dealing with secondary clusters?
	} else {
		err = a.TagResource(clusterResources.VpcID, trimTagPrefix(VpcAvailableTagKey), VpcAvailableTagValueFalse, logger)
		if err != nil {
			return errors.Wrapf(err, "unable to update %s", VpcAvailableTagKey)
		}

		err = a.TagResource(clusterResources.VpcID, trimTagPrefix(VpcClusterIDTagKey), cluster.ID, logger)
		if err != nil {
			return errors.Wrapf(err, "unable to update %s", VpcClusterIDTagKey)
		}

		err = a.TagResource(clusterResources.VpcID, trimTagPrefix(VpcClusterOwnerKey), owner, logger)
		if err != nil {
			return errors.Wrapf(err, "unable to update %s", VpcClusterOwnerKey)
		}
	}

	for _, subnet := range clusterResources.PublicSubnetsIDs {
		err = a.TagResource(subnet, fmt.Sprintf("kubernetes.io/cluster/%s", getClusterTag(cluster)), "shared", logger)
		if err != nil {
			return errors.Wrap(err, "failed to tag subnet")
		}
	}

	for _, callsSecurityGroup := range clusterResources.CallsSecurityGroupIDs {
		err = a.TagResource(callsSecurityGroup, fmt.Sprintf("kubernetes.io/cluster/%s", getClusterTag(cluster)), "shared", logger)
		if err != nil {
			return errors.Wrap(err, "failed to tag subnet")
		}

		err = a.TagResource(callsSecurityGroup, "KubernetesCluster", getClusterTag(cluster), logger)
		if err != nil {
			return errors.Wrap(err, "failed to tag subnet")
		}
	}

	logger.Debugf("Claimed VPC %s", clusterResources.VpcID)

	return nil
}

// releaseVpc
// performs the required tagging to release a VPC from the provided cluster.
// It checks if the cluster is primary or secondary cluster in the VPC, if it is secondary it means
// that a cluster migration is in progress (cluster to cluster on the same VPC).
// If we're removing the secondary cluster, we just cleanup it's tags and we're done.
// If we're removing the primary cluster
//   - and a secondary one is present in the VPC, we remove the references to the primary and
//     promote the secondary to primary.
//   - and there's no secondary cluster, we remove references from the cluster and mark the VPC as
//     available.
//
// If any of the VPC checks either returns no VPCs or more than one VPC this method will fail.
func (a *Client) releaseVpc(cluster *model.Cluster, logger log.FieldLogger) error {
	var isSecondaryCluster bool = false
	secondaryVpcFilters := []ec2Types.Filter{
		{
			Name:   aws.String(VpcAvailableTagKey),
			Values: []string{VpcAvailableTagValueFalse},
		},
		{
			Name:   aws.String(VpcSecondaryClusterIDTagKey),
			Values: []string{cluster.ID},
		},
	}
	vpcs, err := a.GetVpcsWithFilters(secondaryVpcFilters)
	if err != nil {
		return err
	}
	if len(vpcs) != 0 {
		// if this is a secondary cluster in the VPC we need to know to update appropriate tags
		isSecondaryCluster = true
	}

	if !isSecondaryCluster {
		vpcFilters := []ec2Types.Filter{
			{
				Name:   aws.String(VpcAvailableTagKey),
				Values: []string{VpcAvailableTagValueFalse},
			},
			{
				Name:   aws.String(VpcClusterIDTagKey),
				Values: []string{cluster.ID},
			},
		}

		vpcs, err = a.GetVpcsWithFilters(vpcFilters)
		if err != nil {
			return err
		}
	}

	numVPCs := len(vpcs)
	if numVPCs == 0 {
		logger.Warnf("No VPCs are currently claimed by cluster %s, assuming already released", cluster.ID)
		return nil
	}
	if numVPCs != 1 {
		logger.Warn("Multiple VPCs found in release process when expecting 1")
		for i, vpc := range vpcs {
			logger.WithField("tags", vpc.Tags).Warnf("VPC %d: %s", i+1, *vpc.VpcId)
		}
		return fmt.Errorf("multiple VPCs (%d) have been claimed by cluster %s; aborting release process", numVPCs, cluster.ID)
	}

	vpc := vpcs[0]
	publicSubnetFilter := []ec2Types.Filter{
		{
			Name:   aws.String("vpc-id"),
			Values: []string{*vpcs[0].VpcId},
		},
		{
			Name:   aws.String("tag:SubnetType"),
			Values: []string{"public"},
		},
	}

	publicSubnets, err := a.GetSubnetsWithFilters(publicSubnetFilter)
	if err != nil {
		return err
	}

	for _, subnet := range publicSubnets {
		err = a.UntagResource(*subnet.SubnetId, fmt.Sprintf("kubernetes.io/cluster/%s", getClusterTag(cluster)), "shared", logger)
		if err != nil {
			return errors.Wrap(err, "failed to untag subnet")
		}
	}

	callsSecurityGroupFilter := []ec2Types.Filter{
		{
			Name:   aws.String("vpc-id"),
			Values: []string{*vpcs[0].VpcId},
		},
		{
			Name:   aws.String("tag:NodeType"),
			Values: []string{"calls"},
		},
	}

	callsSecurityGroups, err := a.GetSecurityGroupsWithFilters(callsSecurityGroupFilter)
	if err != nil {
		return err
	}

	for _, callsSecurityGroup := range callsSecurityGroups {
		err = a.UntagResource(*callsSecurityGroup.GroupId, fmt.Sprintf("kubernetes.io/cluster/%s", getClusterTag(cluster)), "shared", logger)
		if err != nil {
			return errors.Wrap(err, "failed to tag security group")
		}

		err = a.UntagResource(*callsSecurityGroup.GroupId, "KubernetesCluster", getClusterTag(cluster), logger)
		if err != nil {
			return errors.Wrap(err, "failed to tag security group")
		}
	}

	if isSecondaryCluster {
		err = a.TagResource(*vpc.VpcId, trimTagPrefix(VpcSecondaryClusterIDTagKey), VpcClusterIDTagValueNone, logger)
		if err != nil {
			return errors.Wrapf(err, "unable to update %s", VpcSecondaryClusterIDTagKey)
		}
		logger.Debugf("Secondary cluster %s related tags has been unset from VPC %s", cluster.ID, vpc.VpcId)
		return nil
	}

	if !isSecondaryCluster {
		// If VPC contains a secondary cluster, promote that to primary and stop here, since VPC is not
		// available.
		var secondaryClusterID string
		for _, tag := range vpc.Tags {
			if *tag.Key == trimTagPrefix(VpcSecondaryClusterIDTagKey) {
				secondaryClusterID = *tag.Value
				break
			}
		}

		if secondaryClusterID != VpcClusterIDTagValueNone {
			err = a.TagResource(*vpc.VpcId, trimTagPrefix(VpcClusterIDTagKey), secondaryClusterID, logger)
			if err != nil {
				return errors.Wrapf(err, "unable to update %s", VpcClusterIDTagKey)
			}

			err = a.TagResource(*vpc.VpcId, trimTagPrefix(VpcSecondaryClusterIDTagKey), VpcClusterIDTagValueNone, logger)
			if err != nil {
				return errors.Wrapf(err, "unable to update %s", VpcSecondaryClusterIDTagKey)
			}

			return nil
		}
	}

	err = a.TagResource(*vpc.VpcId, trimTagPrefix(VpcClusterIDTagKey), VpcClusterIDTagValueNone, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to update %s for %s", VpcClusterIDTagKey, *vpc.VpcId)
	}

	err = a.TagResource(*vpc.VpcId, trimTagPrefix(VpcAvailableTagKey), VpcAvailableTagValueTrue, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to update %s for %s", VpcAvailableTagKey, *vpc.VpcId)
	}

	err = a.TagResource(*vpc.VpcId, trimTagPrefix(VpcClusterOwnerKey), VpcClusterOwnerValueNone, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to untag %s from %s", VpcClusterOwnerKey, *vpc.VpcId)
	}

	logger.Debugf("Released VPC %s", *vpc.VpcId)

	return nil
}

// GetVpcResourcesByVpcID retrieve the VPC information for a particulary cluster.
func (a *Client) GetVpcResourcesByVpcID(vpcID string, logger log.FieldLogger) (ClusterResources, error) {
	ctx := context.TODO()
	input := &ec2.DescribeVpcsInput{
		VpcIds: []string{
			vpcID,
		},
	}

	vpcCidr, err := a.Service().ec2.DescribeVpcs(ctx, input)
	if err != nil {
		return ClusterResources{}, errors.Wrapf(err, "failed to fetch the VPC information using VPC ID %s", vpcID)
	}
	return a.getClusterResourcesForVPC(vpcID, *vpcCidr.Vpcs[0].CidrBlock, logger)
}

// TagResourcesByCluster for secondary cluster.
func (a *Client) TagResourcesByCluster(clusterResources ClusterResources, cluster *model.Cluster, owner string, logger log.FieldLogger) error {
	for _, subnet := range clusterResources.PublicSubnetsIDs {
		err := a.TagResource(subnet, fmt.Sprintf("kubernetes.io/cluster/%s", getClusterTag(cluster)), "shared", logger)
		if err != nil {
			return errors.Wrap(err, "failed to tag subnet")
		}
	}

	err := a.TagResource(clusterResources.VpcID, trimTagPrefix(VpcSecondaryClusterIDTagKey), cluster.ID, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to update %s", VpcSecondaryClusterIDTagKey)
	}
	return nil
}

// SwitchClusterTags after migration.
func (a *Client) SwitchClusterTags(clusterID string, targetClusterID string, logger log.FieldLogger) error {
	clusterResources, err := a.GetVpcResources(clusterID, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to get vpc resources for %s", clusterID)
	}

	err = a.TagResource(clusterResources.VpcID, trimTagPrefix(VpcClusterIDTagKey), targetClusterID, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to update %s", VpcClusterIDTagKey)
	}

	err = a.TagResource(clusterResources.VpcID, trimTagPrefix(VpcSecondaryClusterIDTagKey), clusterID, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to update %s", VpcSecondaryClusterIDTagKey)
	}

	return nil
}

func getClusterTag(cluster *model.Cluster) string {
	if cluster.ProvisionerMetadataEKS != nil {
		return eksClusterTag(cluster.ID)
	}
	return kopsClusterTag(cluster.ID)
}

func kopsClusterTag(clusterID string) string {
	return fmt.Sprintf("%s-kops.k8s.local", clusterID)
}
func eksClusterTag(clusterID string) string {
	return clusterID
}
