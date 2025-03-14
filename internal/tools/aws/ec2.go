// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

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

	if AMIImage == "" {
		return true, nil
	}

	if strings.HasPrefix(AMIImage, "ami-") {
		// If AMIImage is an AMI ID, use ImageIds to search for it directly.
		describeIDInput := &ec2.DescribeImagesInput{
			ImageIds: []string{AMIImage},
		}

		output, err := a.Service().ec2.DescribeImages(ctx, describeIDInput)
		if err != nil {
			logger.WithError(err).Error("Failed to describe images by AMI ID")
			return false, err
		}

		if len(output.Images) == 0 {
			logger.Info("No images found matching the AMI ID")
			return false, nil
		}

		return true, nil

	} else {
		// For AMI names, prepare a list of possible AMI names including potential suffixes.
		var amiNames []string

		// If AMIImage already includes an architecture suffix, use it as is.
		if strings.HasSuffix(AMIImage, "-amd64") || strings.HasSuffix(AMIImage, "-arm64") {
			amiNames = []string{AMIImage}
		} else {
			// If AMIImage is a name without an architecture suffix, append "-amd64" and "-arm64".
			amiNames = append(amiNames, AMIImage+"-amd64", AMIImage+"-arm64")
		}

		describeNameInput := &ec2.DescribeImagesInput{
			Filters: []ec2Types.Filter{
				{
					Name:   aws.String("name"),
					Values: amiNames,
				},
			},
		}

		output, err := a.Service().ec2.DescribeImages(ctx, describeNameInput)
		if err != nil {
			logger.WithError(err).Error("Failed to describe images by name")
			return false, err
		}

		if len(output.Images) == 0 {
			logger.Info("No images found matching the criteria", "AMI Names", amiNames)
			return false, nil
		}

		return true, nil
	}
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

func (a *Client) getLaunchTemplate(launchTemplateName string) (*ec2Types.LaunchTemplate, error) {
	launchTemplates, err := a.Service().ec2.DescribeLaunchTemplates(context.TODO(), &ec2.DescribeLaunchTemplatesInput{
		LaunchTemplateNames: []string{launchTemplateName},
	})
	if err != nil {
		if !IsErrorCode(err, "InvalidLaunchTemplateName.NotFoundException") {
			return nil, err
		}
	}

	if launchTemplates == nil || len(launchTemplates.LaunchTemplates) == 0 {
		return nil, nil
	}

	for i, lt := range launchTemplates.LaunchTemplates {
		if *lt.LaunchTemplateName == launchTemplateName {
			return &launchTemplates.LaunchTemplates[i], nil
		}
	}

	a.logger.Debugf("Launch template %s does not exist", launchTemplateName)
	return nil, nil
}

func (a *Client) CreateLaunchTemplate(data *model.LaunchTemplateData) error {
	if data == nil {
		return errors.New("launch template data is nil")
	}

	eksCluster, err := a.getEKSCluster(data.ClusterName)
	if err != nil {
		return errors.Wrap(err, "failed to get eks cluster")
	}

	userData := getLaunchTemplateUserData(eksCluster, data)
	encodedUserData := base64.StdEncoding.EncodeToString([]byte(userData))

	launchTemplate, err := a.Service().ec2.CreateLaunchTemplate(context.TODO(), &ec2.CreateLaunchTemplateInput{
		LaunchTemplateData: &ec2Types.RequestLaunchTemplateData{
			ImageId:          aws.String(data.AMI),
			UserData:         aws.String(encodedUserData),
			SecurityGroupIds: data.SecurityGroups,
		},
		LaunchTemplateName: aws.String(data.Name),
	})
	if err != nil {
		if IsErrorCode(err, "InvalidLaunchTemplateName.AlreadyExistsException") {
			a.logger.Debugf("Launch template %s already exists", data.Name)
			return nil
		}
		return errors.Wrap(err, "failed to create eks launch template")
	}

	if launchTemplate == nil || launchTemplate.LaunchTemplate == nil {
		return errors.New("failed to create eks launch template")
	}

	return nil
}

func (a *Client) UpdateLaunchTemplate(data *model.LaunchTemplateData) error {
	if data == nil {
		return errors.New("launch template data is nil")
	}

	eksCluster, err := a.getEKSCluster(data.ClusterName)
	if err != nil {
		return errors.Wrap(err, "failed to get eks cluster")
	}

	userData := getLaunchTemplateUserData(eksCluster, data)
	encodedUserData := base64.StdEncoding.EncodeToString([]byte(userData))

	launchTemplate, err := a.Service().ec2.CreateLaunchTemplateVersion(context.TODO(), &ec2.CreateLaunchTemplateVersionInput{
		LaunchTemplateData: &ec2Types.RequestLaunchTemplateData{
			ImageId:          aws.String(data.AMI),
			UserData:         aws.String(encodedUserData),
			SecurityGroupIds: data.SecurityGroups,
		},
		LaunchTemplateName: aws.String(data.Name),
	})
	if err != nil {
		if IsErrorCode(err, "InvalidLaunchTemplateName.NotFoundException") {
			return a.CreateLaunchTemplate(data)
		}
		return errors.Wrap(err, "failed to create eks launch template version")
	}

	if launchTemplate == nil || launchTemplate.LaunchTemplateVersion == nil {
		return errors.New("failed to create eks launch template version")
	}

	return nil
}

func (a *Client) DeleteLaunchTemplate(launchTemplateName string) error {
	launchTemplate, err := a.getLaunchTemplate(launchTemplateName)
	if err != nil {
		return err
	}

	if launchTemplate == nil {
		a.logger.Debugf("launch template %s not found, assuming deleted", launchTemplateName)
		return nil
	}

	_, err = a.Service().ec2.DeleteLaunchTemplate(context.TODO(), &ec2.DeleteLaunchTemplateInput{
		LaunchTemplateId: launchTemplate.LaunchTemplateId,
	})
	if err != nil {
		if IsErrorCode(err, "InvalidLaunchTemplateName.NotFoundException") {
			a.logger.Debugf("launch template %s not found, assuming deleted", launchTemplateName)
			return nil
		}
		return errors.Wrap(err, "failed to delete eks launch template")
	}

	return nil
}

func (a *Client) IsLaunchTemplateAvailable(launchTemplateName string) (bool, error) {
	launchTemplate, err := a.getLaunchTemplate(launchTemplateName)
	if err != nil {
		return false, err
	}

	return launchTemplate != nil, nil
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

func getLaunchTemplateUserData(eksCluster *eksTypes.Cluster, data *model.LaunchTemplateData) string {
	dataTemplate := `
#!/bin/bash
set -o xtrace
/etc/eks/bootstrap.sh '%s' --apiserver-endpoint '%s' --b64-cluster-ca '%s' --use-max-pods false  --kubelet-extra-args '--max-pods=%d'`
	return fmt.Sprintf(dataTemplate, *eksCluster.Name, *eksCluster.Endpoint, *eksCluster.CertificateAuthority.Data, data.MaxPodsPerNode)
}

func (a *Client) GetAMIByTag(tagKey, tagValue string, logger log.FieldLogger) (string, error) {
	ctx := context.TODO()

	images, err := a.Service().ec2.DescribeImages(ctx, &ec2.DescribeImagesInput{
		Owners: []string{"self"},
		Filters: []ec2Types.Filter{
			{
				Name:   aws.String(fmt.Sprintf("tag:%s", tagKey)),
				Values: []string{tagValue},
			},
		},
	})
	if err != nil {
		return "", errors.Wrap(err, "failed to describe images by tag")
	}

	if len(images.Images) == 0 {
		a.logger.Info("No images found matching the criteria.")
		return "", nil
	}

	//Sort images by creation date
	sort.Slice(images.Images, func(i, j int) bool {
		return *images.Images[i].CreationDate > *images.Images[j].CreationDate
	})

	return *images.Images[0].ImageId, nil
}
