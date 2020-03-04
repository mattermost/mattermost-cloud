package aws

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// ClusterResources is a collection of AWS resources that will be used to create
// a kops cluster.
type ClusterResources struct {
	VpcID                  string
	PrivateSubnetIDs       []string
	PublicSubnetsIDs       []string
	MasterSecurityGroupIDs []string
	WorkerSecurityGroupIDs []string
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

	return nil
}

func (a *Client) getClusterResourcesForVPC(vpcID string) (ClusterResources, error) {
	clusterResources := ClusterResources{
		VpcID: vpcID,
	}

	baseFilter := []*ec2.Filter{
		{
			Name:   aws.String("vpc-id"),
			Values: []*string{aws.String(vpcID)},
		},
	}

	privateSubnetFilter := append(baseFilter, &ec2.Filter{
		Name:   aws.String("tag:SubnetType"),
		Values: []*string{aws.String("private")},
	})

	privateSubnets, err := a.GetSubnetsWithFilters(privateSubnetFilter)
	if err != nil {
		return clusterResources, err
	}

	for _, subnet := range privateSubnets {
		clusterResources.PrivateSubnetIDs = append(clusterResources.PrivateSubnetIDs, *subnet.SubnetId)
	}

	publicSubnetFilter := append(baseFilter, &ec2.Filter{
		Name:   aws.String("tag:SubnetType"),
		Values: []*string{aws.String("public")},
	})

	publicSubnets, err := a.GetSubnetsWithFilters(publicSubnetFilter)
	if err != nil {
		return clusterResources, err
	}

	for _, subnet := range publicSubnets {
		clusterResources.PublicSubnetsIDs = append(clusterResources.PublicSubnetsIDs, *subnet.SubnetId)
	}

	masterSGFilter := append(baseFilter, &ec2.Filter{
		Name:   aws.String("tag:NodeType"),
		Values: []*string{aws.String("master")},
	})

	masterSecurityGroups, err := a.GetSecurityGroupsWithFilters(masterSGFilter)
	if err != nil {
		return clusterResources, err
	}

	for _, securityGroup := range masterSecurityGroups {
		clusterResources.MasterSecurityGroupIDs = append(clusterResources.MasterSecurityGroupIDs, *securityGroup.GroupId)
	}

	workerSGFilter := append(baseFilter, &ec2.Filter{
		Name:   aws.String("tag:NodeType"),
		Values: []*string{aws.String("worker")},
	})

	workerSecurityGroups, err := a.GetSecurityGroupsWithFilters(workerSGFilter)
	if err != nil {
		return clusterResources, err
	}

	for _, securityGroup := range workerSecurityGroups {
		clusterResources.WorkerSecurityGroupIDs = append(clusterResources.WorkerSecurityGroupIDs, *securityGroup.GroupId)
	}

	err = clusterResources.IsValid()
	if err != nil {
		return clusterResources, errors.Wrapf(err, "VPC %s is misconfigured", clusterResources.VpcID)
	}

	return clusterResources, nil
}

// GetAndClaimVpcResources creates ClusterResources from an available VPC and
// tags them appropriately.
func (a *Client) GetAndClaimVpcResources(clusterID, owner string, logger log.FieldLogger) (ClusterResources, error) {
	// First, check if a VPC has been claimed by this cluster. If only one has
	// already been claimed, then return that with no error.
	clusterAlreadyClaimedFilter := []*ec2.Filter{
		{
			Name: aws.String(VpcAvailableTagKey),
			Values: []*string{
				aws.String(VpcAvailableTagValueFalse),
			},
		},
		{
			Name: aws.String(VpcClusterIDTagKey),
			Values: []*string{
				aws.String(clusterID),
			},
		},
	}
	clusterAlreadyClaimedVpcs, err := a.GetVpcsWithFilters(clusterAlreadyClaimedFilter)
	if err != nil {
		return ClusterResources{}, err
	}
	if len(clusterAlreadyClaimedVpcs) > 1 {
		return ClusterResources{}, fmt.Errorf("multiple VPCs (%d) have been claimed by cluster %s; aborting claim process", len(clusterAlreadyClaimedVpcs), clusterID)
	}
	if len(clusterAlreadyClaimedVpcs) == 1 {
		return a.getClusterResourcesForVPC(*clusterAlreadyClaimedVpcs[0].VpcId)
	}

	// This cluster has not already claimed a VPC. Continue with claiming process.
	totalVpcsFilter := []*ec2.Filter{
		{
			Name: aws.String(VpcAvailableTagKey),
			Values: []*string{
				aws.String(VpcAvailableTagValueTrue),
				aws.String(VpcAvailableTagValueFalse),
			},
		},
	}
	totalVpcs, err := a.GetVpcsWithFilters(totalVpcsFilter)
	if err != nil {
		return ClusterResources{}, err
	}
	totalVpcCount := len(totalVpcs)

	vpcFilters := []*ec2.Filter{
		{
			Name:   aws.String(VpcAvailableTagKey),
			Values: []*string{aws.String(VpcAvailableTagValueTrue)},
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
		clusterResources, err := a.getClusterResourcesForVPC(*vpc.VpcId)
		if err != nil {
			logger.Warn(err)
			continue
		}

		err = a.claimVpc(clusterResources, clusterID, owner, logger)
		if err != nil {
			return clusterResources, err
		}

		return clusterResources, nil
	}

	return ClusterResources{}, fmt.Errorf("%d VPCs were returned as currently available; none of them were configured correctly", len(vpcs))
}

// ReleaseVpc changes the tags on a VPC to mark it as "available" again.
func (a *Client) ReleaseVpc(clusterID string, logger log.FieldLogger) error {
	return a.releaseVpc(clusterID, logger)
}

// claimVpc will claim the given VPC for a cluster if a final race-check passes.
// The final race check does the following:
//   - Requires the VPC to exist. #mindblown
//   - VPC availabiltiy tag must be "true"
//   - VPC cluster ID tag must by "none"
func (a *Client) claimVpc(clusterResources ClusterResources, clusterID string, owner string, logger log.FieldLogger) error {
	vpcFilter := []*ec2.Filter{
		{
			Name:   aws.String("vpc-id"),
			Values: []*string{aws.String(clusterResources.VpcID)},
		},
		{
			Name:   aws.String(VpcAvailableTagKey),
			Values: []*string{aws.String(VpcAvailableTagValueTrue)},
		},
		{
			Name:   aws.String(VpcClusterIDTagKey),
			Values: []*string{aws.String(VpcClusterIDTagValueNone)},
		},
	}
	vpcs, err := a.GetVpcsWithFilters(vpcFilter)
	if err != nil {
		return err
	}

	numVPCs := len(vpcs)
	if numVPCs == 0 {
		return fmt.Errorf("query didn't return VPC %s; it either doesn't exist or another cluster claimed it", clusterResources.VpcID)
	}
	if numVPCs != 1 {
		return fmt.Errorf("query for VPC %s somehow returned multiple results", clusterResources.VpcID)
	}

	err = a.TagResource(clusterResources.VpcID, trimTagPrefix(VpcAvailableTagKey), VpcAvailableTagValueFalse, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to update %s", VpcAvailableTagKey)
	}

	err = a.TagResource(clusterResources.VpcID, trimTagPrefix(VpcClusterIDTagKey), clusterID, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to update %s", VpcClusterIDTagKey)
	}

	err = a.TagResource(clusterResources.VpcID, trimTagPrefix(VpcClusterOwnerKey), owner, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to update %s", VpcClusterIDTagKey)
	}

	for _, subnet := range clusterResources.PublicSubnetsIDs {
		err = a.TagResource(subnet, fmt.Sprintf("kubernetes.io/cluster/%s", fmt.Sprintf("%s-kops.k8s.local", clusterID)), "shared", logger)
		if err != nil {
			return errors.Wrap(err, "failed to tag subnet")
		}
	}

	logger.Debugf("Claimed VPC %s", clusterResources.VpcID)

	return nil
}

func (a *Client) releaseVpc(clusterID string, logger log.FieldLogger) error {
	vpcFilters := []*ec2.Filter{
		{
			Name:   aws.String(VpcAvailableTagKey),
			Values: []*string{aws.String(VpcAvailableTagValueFalse)},
		},
		{
			Name:   aws.String(VpcClusterIDTagKey),
			Values: []*string{aws.String(clusterID)},
		},
	}

	vpcs, err := a.GetVpcsWithFilters(vpcFilters)
	if err != nil {
		return err
	}

	numVPCs := len(vpcs)
	if numVPCs == 0 {
		logger.Warnf("No VPCs are currently claimed by cluster %s, assuming already released", clusterID)
		return nil
	}
	if numVPCs != 1 {
		logger.Warn("Multiple VPCs found in release process when expecting 1")
		for i, vpc := range vpcs {
			logger.WithField("tags", vpc.Tags).Warnf("VPC %d: %s", i+1, *vpc.VpcId)
		}
		return fmt.Errorf("multiple VPCs (%d) have been claimed by cluster %s; aborting release process", numVPCs, clusterID)
	}

	publicSubnetFilter := []*ec2.Filter{
		{
			Name:   aws.String("vpc-id"),
			Values: []*string{vpcs[0].VpcId},
		},
		{
			Name:   aws.String("tag:SubnetType"),
			Values: []*string{aws.String("public")},
		},
	}

	publicSubnets, err := a.GetSubnetsWithFilters(publicSubnetFilter)
	if err != nil {
		return err
	}

	for _, subnet := range publicSubnets {
		err = a.UntagResource(*subnet.SubnetId, fmt.Sprintf("kubernetes.io/cluster/%s", fmt.Sprintf("%s-kops.k8s.local", clusterID)), "shared", logger)
		if err != nil {
			return errors.Wrap(err, "failed to untag subnet")
		}
	}

	err = a.TagResource(*vpcs[0].VpcId, trimTagPrefix(VpcClusterIDTagKey), VpcClusterIDTagValueNone, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to update %s", VpcClusterIDTagKey)
	}

	err = a.TagResource(*vpcs[0].VpcId, trimTagPrefix(VpcAvailableTagKey), VpcAvailableTagValueTrue, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to update %s", VpcAvailableTagKey)
	}

	err = a.TagResource(*vpcs[0].VpcId, trimTagPrefix(VpcClusterOwnerKey), VpcClusterOwnerValueNone, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to untag owner from %s", *vpcs[0].VpcId)
	}

	logger.Debugf("Released VPC %s", *vpcs[0].VpcId)

	return nil
}
