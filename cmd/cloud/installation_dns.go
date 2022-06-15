// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {

	installationDNSAddCmd.Flags().String("installation", "", "The id of the installation to add domain to.")
	installationDNSAddCmd.MarkFlagRequired("installation")
	installationDNSAddCmd.Flags().String("domain", "", "Domain name to map to the installation.")
	installationDNSAddCmd.MarkFlagRequired("domain")

	installationDNSSetPrimaryCmd.Flags().String("installation", "", "The id of the installation for domain switch.")
	installationDNSSetPrimaryCmd.MarkFlagRequired("installation")
	installationDNSSetPrimaryCmd.Flags().String("domain-id", "", "The id of domain name to set as primary.")
	installationDNSSetPrimaryCmd.MarkFlagRequired("domain-id")

	installationDNSCmd.AddCommand(installationDNSAddCmd)
	installationDNSCmd.AddCommand(installationDNSSetPrimaryCmd)
}

var installationDNSCmd = &cobra.Command{
	Use:   "dns",
	Short: "Manipulate installation DNS records.",
}

var installationDNSAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Adds domain name for the installation.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		dnsName, _ := command.Flags().GetString("domain")

		request := &model.AddDNSRecordRequest{
			DNS: dnsName,
		}

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			err := printJSON(request)
			if err != nil {
				return errors.Wrap(err, "failed to print API request")
			}

			return nil
		}

		installation, err := client.AddInstallationDNS(installationID, request)
		if err != nil {
			return errors.Wrap(err, "failed to add installation dns")
		}

		err = printJSON(installation)
		if err != nil {
			return errors.Wrap(err, "failed to print response")
		}

		return nil
	},
}

var installationDNSSetPrimaryCmd = &cobra.Command{
	Use:   "set-primary",
	Short: "Sets installation domain name as primary.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		domainNameID, _ := command.Flags().GetString("domain-id")

		installation, err := client.SetInstallationDomainPrimary(installationID, domainNameID)
		if err != nil {
			return errors.Wrap(err, "failed to set installation domain primary")
		}

		err = printJSON(installation)
		if err != nil {
			return errors.Wrap(err, "failed to print response")
		}

		return nil
	},
}
