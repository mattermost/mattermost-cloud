// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import "github.com/spf13/cobra"

type clusterNodegroupsAddFlags struct {
	clusterFlags
	clusterID                   string
	nodegroups                  map[string]string
	nodegroupsWithPublicSubnet  []string
	nodegroupsWithSecurityGroup []string
}

func (flags *clusterNodegroupsAddFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.clusterID, "cluster", "", "The id of the cluster to be modified.")
	command.Flags().StringToStringVar(&flags.nodegroups, "nodegroups", nil, "Additional nodegroups to create. The key is the name of the nodegroup and the value is the size constant.")
	command.Flags().StringSliceVar(&flags.nodegroupsWithPublicSubnet, "nodegroups-with-public-subnet", nil, "Nodegroups to create with public subnet. The value is the name of the nodegroup.")
	command.Flags().StringSliceVar(&flags.nodegroupsWithSecurityGroup, "nodegroups-with-sg", nil, "Nodegroups to create with dedicated security group. The value is the name of the nodegroup.")

	_ = command.MarkFlagRequired("cluster")
	_ = command.MarkFlagRequired("nodegroups")
}
