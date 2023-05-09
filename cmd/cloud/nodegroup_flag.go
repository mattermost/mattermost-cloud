// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import "github.com/spf13/cobra"

type clusterNodegroupsCreateFlags struct {
	clusterFlags
	clusterID                   string
	nodegroups                  map[string]string
	nodegroupsWithPublicSubnet  []string
	nodegroupsWithSecurityGroup []string
}

func (flags *clusterNodegroupsCreateFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.clusterID, "cluster", "", "The id of the cluster to be modified.")
	command.Flags().StringToStringVar(&flags.nodegroups, "nodegroups", nil, "Additional nodegroups to create. Example: --nodegroups ng1=t3.medium;min=1;max=3")
	command.Flags().StringSliceVar(&flags.nodegroupsWithPublicSubnet, "nodegroups-with-public-subnet", nil, "Nodegroups to create with public subnet. The value is the name of the nodegroups.")
	command.Flags().StringSliceVar(&flags.nodegroupsWithSecurityGroup, "nodegroups-with-sg", nil, "Nodegroups to create with dedicated security group. The value is the name of the nodegroupa.")

	_ = command.MarkFlagRequired("cluster")
	_ = command.MarkFlagRequired("nodegroups")
}

type clusterNodegroupDeleteFlags struct {
	clusterFlags
	clusterID string
	nodegroup string
}

func (flags *clusterNodegroupDeleteFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.clusterID, "cluster", "", "The id of the cluster to be modified.")
	command.Flags().StringVar(&flags.nodegroup, "nodegroup", "", "The name of the nodegroup to delete.")

	_ = command.MarkFlagRequired("cluster")
	_ = command.MarkFlagRequired("nodegroup")
}
