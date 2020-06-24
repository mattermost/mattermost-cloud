// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/internal/tools/terraform"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	workbenchCmd.PersistentFlags().String("server", "http://localhost:8075", "The provisioning server whose API will be queried.")
	workbenchCmd.PersistentFlags().String("state-store", "dev.cloud.mattermost.com", "The S3 bucket used to store cluster state.")

	workbenchClusterCmd.Flags().String("cluster", "", "The id of the cluster to work on.")
	workbenchClusterCmd.MarkFlagRequired("cluster")

	workbenchCmd.AddCommand(workbenchClusterCmd)
}

var workbenchCmd = &cobra.Command{
	Use:   "workbench",
	Short: "Tools for working with cloud resources",
}

var workbenchClusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Export kops and terraform files into a workbench directory",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterID, _ := command.Flags().GetString("cluster")
		cluster, err := client.GetCluster(clusterID)
		if err != nil {
			return errors.Wrap(err, "failed to query cluster")
		}
		if cluster == nil {
			return errors.Errorf("unable to find cluster %s", clusterID)
		}

		logger := logger.WithField("cluster", clusterID)
		logger.Info("Setting up cluster workbench")

		s3StateStore, _ := command.Flags().GetString("state-store")

		kopsClient, err := kops.New(s3StateStore, logger)
		if err != nil {
			return err
		}

		err = kopsClient.ExportKubecfg(cluster.ProvisionerMetadataKops.Name)
		if err != nil {
			kopsClient.Close()
			return err
		}

		err = kopsClient.UpdateCluster(cluster.ProvisionerMetadataKops.Name, kopsClient.GetOutputDirectory())
		if err != nil {
			kopsClient.Close()
			return err
		}

		terraformClient, err := terraform.New(kopsClient.GetOutputDirectory(), s3StateStore, logger)
		if err != nil {
			kopsClient.Close()
			return err
		}

		err = terraformClient.Init(cluster.ProvisionerMetadataKops.Name)
		if err != nil {
			kopsClient.Close()
			terraformClient.Close()
			return err
		}

		logger.Info("Cluster workbench setup complete")
		logger.Infof("The workbench directory can be found at %s", kopsClient.GetTempDir())

		return nil
	},
}
