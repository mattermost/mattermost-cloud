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

func (flags *installationDNSAddFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to add domain to.")
	command.Flags().StringVar(&flags.dnsName, "domain", "", "Domain name to map to the installation.")
	_ = command.MarkFlagRequired("installation")
	_ = command.MarkFlagRequired("domain")
}

type installationDNSSetPrimaryFlags struct {
	clusterFlags
	installationID string
	domainNameID   string
}

func (flags *installationDNSSetPrimaryFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation for domain switch.")
	command.Flags().StringVar(&flags.domainNameID, "domain-id", "", "The id of domain name to set as primary.")
	_ = command.MarkFlagRequired("installation")
	_ = command.MarkFlagRequired("domain-id")

}
