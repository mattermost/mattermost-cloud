// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newCmdInstallationDNS() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dns",
		Short: "Manipulate installation DNS records.",
	}

	cmd.AddCommand(newCmdInstallationDNSAdd())
	cmd.AddCommand(newCmdInstallationDNSSetPrimary())

	return cmd
}

func newCmdInstallationDNSAdd() *cobra.Command {
	var flags installationDNSAddFlags

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Adds domain name for the installation.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := model.NewClient(flags.serverAddress)

			request := &model.AddDNSRecordRequest{
				DNS: flags.dnsName,
			}

			if flags.dryRun {
				return runDryRun(request)
			}

			installation, err := client.AddInstallationDNS(flags.installationID, request)
			if err != nil {
				return errors.Wrap(err, "failed to add installation dns")
			}

			if err = printJSON(installation); err != nil {
				return errors.Wrap(err, "failed to print response")
			}

			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdInstallationDNSSetPrimary() *cobra.Command {
	var flags installationDNSSetPrimaryFlags

	cmd := &cobra.Command{
		Use:   "set-primary",
		Short: "Sets installation domain name as primary.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := model.NewClient(flags.serverAddress)

			installation, err := client.SetInstallationDomainPrimary(flags.installationID, flags.domainNameID)
			if err != nil {
				return errors.Wrap(err, "failed to set installation domain primary")
			}

			if err = printJSON(installation); err != nil {
				return errors.Wrap(err, "failed to print response")
			}

			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}
