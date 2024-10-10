// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"context"
	"fmt"

	"github.com/mattermost/mattermost-cloud/clusterdictionary"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newCmdClusterNodegroup() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodegroup",
		Short: "Manipulate nodegroups of an existing cluster.",
	}

	cmd.AddCommand(newCmdClusterNodegroupCreate())
	cmd.AddCommand(newCmdClusterNodegroupDelete())

	return cmd
}

func newCmdClusterNodegroupCreate() *cobra.Command {
	var flags clusterNodegroupsCreateFlags
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create new nodegroups in an existing cluster.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return addNodegroup(command.Context(), flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func addNodegroup(ctx context.Context, flags clusterNodegroupsCreateFlags) error {
	client := createClient(ctx, flags.clusterFlags)

	if len(flags.nodegroups) == 0 {
		return fmt.Errorf("nodegroups must be provided")
	}

	for _, ng := range flags.nodegroupsWithPublicSubnet {
		if _, f := flags.nodegroups[ng]; !f {
			return fmt.Errorf("nodegroup %s not provided in nodegroups", ng)
		}
	}

	for _, ng := range flags.nodegroupsWithSecurityGroup {
		if _, f := flags.nodegroups[ng]; !f {
			return fmt.Errorf("nodegroup %s not provided in nodegroups", ng)
		}
	}

	request := model.CreateNodegroupsRequest{
		NodeGroupWithPublicSubnet:  flags.nodegroupsWithPublicSubnet,
		NodeGroupWithSecurityGroup: flags.nodegroupsWithSecurityGroup,
	}

	err := clusterdictionary.AddToCreateNodegroupsRequest(flags.nodegroups, &request)
	if err != nil {
		return errors.Wrap(err, "failed to apply size values for nodegroups")
	}

	if flags.dryRun {
		return runDryRun(request)
	}

	cluster, err := client.CreateNodegroups(flags.clusterID, &request)
	if err != nil {
		return errors.Wrap(err, "failed to create cluster")
	}

	if err = printJSON(cluster); err != nil {
		return errors.Wrap(err, "failed to print cluster response")
	}

	return nil
}

func newCmdClusterNodegroupDelete() *cobra.Command {
	var flags clusterNodegroupDeleteFlags

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a nodegroup from an existing cluster.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return deleteNodegroup(command.Context(), flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func deleteNodegroup(ctx context.Context, flags clusterNodegroupDeleteFlags) error {
	client := createClient(ctx, flags.clusterFlags)

	if len(flags.nodegroup) == 0 {
		return fmt.Errorf("nodegroup must be provided")
	}

	cluster, err := client.DeleteNodegroup(flags.clusterID, flags.nodegroup)
	if err != nil {
		return errors.Wrap(err, "failed to delete nodegroup")
	}

	if err = printJSON(cluster); err != nil {
		return errors.Wrap(err, "failed to print cluster response")
	}

	return nil
}
