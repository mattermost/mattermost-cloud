// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import "github.com/spf13/cobra"

type installationDNSAddFlags struct {
	clusterFlags
	installationID string
	dnsName        string
}

func (flags *installationDNSAddFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to add domain to.")
	cmd.Flags().StringVar(&flags.dnsName, "domain", "", "Domain name to map to the installation.")
	_ = cmd.MarkFlagRequired("installation")
	_ = cmd.MarkFlagRequired("domain")
}

type installationDNSSetPrimaryFlags struct {
	clusterFlags
	installationID string
	domainNameID   string
}

func (flags *installationDNSSetPrimaryFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation for domain switch.")
	cmd.Flags().StringVar(&flags.domainNameID, "domain-id", "", "The id of domain name to set as primary.")
	_ = cmd.MarkFlagRequired("installation")
	_ = cmd.MarkFlagRequired("domain-id")

}
