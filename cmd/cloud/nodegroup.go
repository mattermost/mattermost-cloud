// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"fmt"

	"github.com/mattermost/mattermost-cloud/clusterdictionary"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newCmdClusterNodegroups() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "nodegroups",
		Short: "Manipulate nodegroups of an existing cluster.",
	}

	cmd.AddCommand(newCmdClusterNodegroupsAdd())

	return cmd
}

func newCmdClusterNodegroupsAdd() *cobra.Command {
	var flags clusterNodegroupsAddFlags
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create new nodegroups in an existing cluster.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return addNodegroup(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func addNodegroup(flags clusterNodegroupsAddFlags) error {
	client := model.NewClient(flags.serverAddress)

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
