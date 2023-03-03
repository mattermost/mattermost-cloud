// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	ec2Types "github.com/aws/aws-sdk-go-v2/service/ec2/types"
	eksTypes "github.com/aws/aws-sdk-go-v2/service/eks/types"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// TagResource tags an AWS EC2 resource.
func (a *Client) TagResource(resourceID, key, value string, logger log.FieldLogger) error {
	ctx := context.TODO()

	if resourceID == "" {
		return errors.New("Missing resource ID")
	}

	output, err := a.Service().ec2.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{resourceID},
		Tags: []ec2Types.Tag{
			{
				Key:   aws.String(key),
				Value: aws.String(value),
			},
		},
	})
	if err != nil {
		return errors.Wrapf(err, "unable to tag resource id: %s", resourceID)
	}

	logger.WithFields(log.Fields{
		"tag-key":   key,
		"tag-value": value,
	}).Debugf("AWS EC2 create tag response for %s: %s", resourceID, prettyCreateTagsResponse(output))

	return nil
}

// UntagResource deletes tags from an AWS EC2 resource.
func (a *Client) UntagResource(resourceID, key, value string, logger log.FieldLogger) error {
	ctx := context.TODO()

	if resourceID == "" {
		return errors.New("unable to remove AWS tag from resource: missing resource ID")
	}

	output, err := a.Service().ec2.DeleteTags(ctx, &ec2.DeleteTagsInput{
		Resources: []string{
			resourceID,
		},
		Tags: []ec2Types.Tag{
			{
				Key:   aws.String(key),
				Value: aws.String(value),
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "unable to remove AWS tag from resource")
	}

	logger.WithFields(log.Fields{
		"tag-key":   key,
		"tag-value": value,
	}).Debugf("AWS EC2 delete tag response for %s: %s", resourceID, prettyDeleteTagsResponse(output))

	return nil
}

// IsValidAMI check if the provided AMI exists
func (a *Client) IsValidAMI(AMIImage string, logger log.FieldLogger) (bool, error) {
	ctx := context.TODO()

	// if AMI image is blank it will use the default KOPS image
	if AMIImage == "" {
		return true, nil
	}

	output, err := a.Service().ec2.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String("image-id"),
				Values: []string{AMIImage},
			},
		},
	})
	if err != nil {
		return false, err
	}
	if len(output.Images) == 0 {
		return false, nil
	}

	return true, nil
}

// GetVpcsWithFilters returns VPCs matching a given filter.
func (a *Client) GetVpcsWithFilters(filters []ec2Types.Filter) ([]ec2Types.Vpc, error) {
	ctx := context.TODO()

	output, err := a.Service().ec2.DescribeVpcs(ctx, &ec2.DescribeVpcsInput{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	return output.Vpcs, nil
}

// GetSubnetsWithFilters returns subnets matching a given filter.
func (a *Client) GetSubnetsWithFilters(filters []ec2Types.Filter) ([]ec2Types.Subnet, error) {
	ctx := context.TODO()

	output, err := a.Service().ec2.DescribeSubnets(ctx, &ec2.DescribeSubnetsInput{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	return output.Subnets, nil
}

// GetSecurityGroupsWithFilters returns SGs matching a given filter.
func (a *Client) GetSecurityGroupsWithFilters(filters []ec2Types.Filter) ([]ec2Types.SecurityGroup, error) {
	ctx := context.TODO()

	output, err := a.Service().ec2.DescribeSecurityGroups(ctx, &ec2.DescribeSecurityGroupsInput{
		Filters: filters,
	})
	if err != nil {
		return nil, err
	}

	return output.SecurityGroups, nil
}

func getLaunchTemplateName(clusterName string) string {
	return fmt.Sprintf("%s-launch-template", clusterName)
}

func (a *Client) getLaunchTemplate(clusterName string) (*ec2Types.LaunchTemplate, error) {
	launchTemplateName := getLaunchTemplateName(clusterName)

	launchTemplates, err := a.Service().ec2.DescribeLaunchTemplates(context.TODO(), &ec2.DescribeLaunchTemplatesInput{
		LaunchTemplateNames: []string{launchTemplateName},
	})
	if err != nil {
		return nil, err
	}

	for i, lt := range launchTemplates.LaunchTemplates {
		if *lt.LaunchTemplateName == launchTemplateName {
			return &launchTemplates.LaunchTemplates[i], nil
		}
	}

	return nil, nil
}

func (a *Client) EnsureLaunchTemplate(clusterName string, eksMetadata *model.EKSMetadata) (*int64, error) {

	launchTemplate, err := a.getLaunchTemplate(clusterName)
	if err != nil {
		if !IsErrorCode(err, "InvalidLaunchTemplateName.NotFoundException") {
			return nil, errors.Wrap(err, "failed to get launch template")
		}
	}

	if launchTemplate != nil {
		return launchTemplate.LatestVersionNumber, nil
	}

	return a.createLaunchTemplate(clusterName, eksMetadata)

}

func (a *Client) createLaunchTemplate(clusterName string, eksMetadata *model.EKSMetadata) (*int64, error) {
	eksCluster, err := a.getEKSCluster(clusterName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get eks cluster")
	}

	userData := getLaunchTemplateUserData(eksCluster, eksMetadata)
	encodedUserData := base64.StdEncoding.EncodeToString([]byte(userData))

	launchTemplate, err := a.Service().ec2.CreateLaunchTemplate(context.TODO(), &ec2.CreateLaunchTemplateInput{
		LaunchTemplateData: &ec2Types.RequestLaunchTemplateData{
			ImageId:  aws.String(eksMetadata.ChangeRequest.AMI),
			UserData: aws.String(encodedUserData),
		},
		LaunchTemplateName: aws.String(getLaunchTemplateName(clusterName)),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create eks launch template")
	}

	return launchTemplate.LaunchTemplate.LatestVersionNumber, nil
}

func (a *Client) UpdateLaunchTemplate(clusterName string, eksMetadata *model.EKSMetadata) (*int64, error) {
	eksCluster, err := a.getEKSCluster(clusterName)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get eks cluster")
	}

	if eksMetadata.ChangeRequest == nil {
		eksMetadata.ChangeRequest = &model.EKSMetadataRequestedState{}
	}

	if eksMetadata.ChangeRequest.AMI == "" {
		eksMetadata.ChangeRequest.AMI = eksMetadata.AMI
	}
	if eksMetadata.ChangeRequest.MaxPodsPerNode == 0 {
		eksMetadata.ChangeRequest.MaxPodsPerNode = eksMetadata.MaxPodsPerNode
	}

	userData := getLaunchTemplateUserData(eksCluster, eksMetadata)
	encodedUserData := base64.StdEncoding.EncodeToString([]byte(userData))

	launchTemplate, err := a.Service().ec2.CreateLaunchTemplateVersion(context.TODO(), &ec2.CreateLaunchTemplateVersionInput{
		LaunchTemplateData: &ec2Types.RequestLaunchTemplateData{
			ImageId:  aws.String(eksMetadata.ChangeRequest.AMI),
			UserData: aws.String(encodedUserData),
		},
		LaunchTemplateName: aws.String(getLaunchTemplateName(clusterName)),
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create eks launch template")
	}

	return launchTemplate.LaunchTemplateVersion.VersionNumber, nil
}

func (a *Client) EnsureLaunchTemplateDeleted(clusterName string) error {
	launchTemplate, err := a.getLaunchTemplate(clusterName)
	if err != nil {
		if IsErrorCode(err, "InvalidLaunchTemplateName.NotFoundException") {
			return nil
		}
		return err
	}

	if launchTemplate == nil {
		return nil
	}

	_, err = a.Service().ec2.DeleteLaunchTemplate(context.TODO(), &ec2.DeleteLaunchTemplateInput{
		LaunchTemplateId: launchTemplate.LaunchTemplateId,
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete eks launch template")
	}

	return nil
}

func prettyCreateTagsResponse(output *ec2.CreateTagsOutput) string {
	prettyResp, err := json.Marshal(output)
	if err != nil {
		return fmt.Sprintf("%v", output)
	}

	return string(prettyResp)
}

func prettyDeleteTagsResponse(output *ec2.DeleteTagsOutput) string {
	prettyResp, err := json.Marshal(output)
	if err != nil {
		return fmt.Sprintf("%v", output)
	}

	return string(prettyResp)
}

func getLaunchTemplateUserData(eksCluster *eksTypes.Cluster, eksMeta *model.EKSMetadata) string {
	dataTemplate := `
#!/bin/bash
set -o xtrace
/etc/eks/bootstrap.sh '%s' --apiserver-endpoint '%s' --b64-cluster-ca '%s' --use-max-pods false  --kubelet-extra-args '--max-pods=%d'`
	return fmt.Sprintf(dataTemplate, *eksCluster.Name, *eksCluster.Endpoint, *eksCluster.CertificateAuthority.Data, eksMeta.ChangeRequest.MaxPodsPerNode)
}
