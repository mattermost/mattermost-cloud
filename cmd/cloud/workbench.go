// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"context"

	awsTools "github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/internal/tools/terraform"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newCmdWorkbench() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "workbench",
		Short: "Tools for working with cloud resources",
	}

	setWorkbenchFlags(cmd)

	cmd.AddCommand(newCmdWorkbenchCluster())

	return cmd
}

func newCmdWorkbenchCluster() *cobra.Command {

	var flags workbenchClusterFlag

	cmd := &cobra.Command{
		Use:   "cluster",
		Short: "Export kops and terraform files into a workbench directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return executeWorkbenchClusterCmd(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.workbenchFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func executeWorkbenchClusterCmd(flags workbenchClusterFlag) error {
	client := model.NewClient(flags.serverAddress)
	cluster, err := client.GetCluster(flags.clusterID)
	if err != nil {
		return errors.Wrap(err, "failed to query cluster")
	}
	if cluster == nil {
		return errors.Errorf("unable to find cluster %s", flags.clusterID)
	}
	awsConfig, err := awsTools.NewAWSConfig(context.TODO())
	if err != nil {
		return errors.Wrap(err, "failed to get aws configuration")
	}
	awsClient, err := awsTools.NewAWSClientWithConfig(&awsConfig, logger)
	if err != nil {
		return errors.Wrap(err, "failed to build AWS client")
	}

	logger := logger.WithField("cluster", flags.clusterID)
	logger.Info("Setting up cluster workbench")

	kopsClient, err := kops.New(flags.s3StateStore, logger)
	if err != nil {
		return err
	}

	if err = kopsClient.ExportKubecfg(cluster.ProvisionerMetadataKops.Name); err != nil {
		return err
	}

	if err = kopsClient.UpdateCluster(cluster.ProvisionerMetadataKops.Name, kopsClient.GetOutputDirectory()); err != nil {
		return err
	}

	err = awsClient.FixSubnetTagsForVPC(cluster.ProvisionerMetadataKops.VPC, logger)
	if err != nil {
		return err
	}

	terraformClient, err := terraform.New(kopsClient.GetOutputDirectory(), flags.s3StateStore, logger)
	if err != nil {
		return err
	}
	defer func() {
		if errDefer := terraformClient.Close(); errDefer != nil {
			logger.WithError(errDefer).Error("Failed to close terraform client")
		}
	}()

	if err = terraformClient.Init(cluster.ProvisionerMetadataKops.Name); err != nil {
		return err
	}

	logger.Info("Cluster workbench setup complete")
	logger.Infof("The workbench directory can be found at %s", kopsClient.GetTempDir())

	return nil
}
