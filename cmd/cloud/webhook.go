// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"os"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func webhookCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Manipulate webhooks managed by the provisioning server.",
	}

	setWebhookFlags(cmd)

	cmd.AddCommand(webhookCreateCmd())
	cmd.AddCommand(webhookGetCmd())
	cmd.AddCommand(webhookListCmd())
	cmd.AddCommand(webhookDeleteCmd())

	return cmd
}

func webhookCreateCmd() *cobra.Command {
	var flags webhookCreateFlag

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a webhook.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)

			webhook, err := client.CreateWebhook(&model.CreateWebhookRequest{
				OwnerID: flags.ownerID,
				URL:     flags.url,
			})
			if err != nil {
				return errors.Wrap(err, "failed to create webhook")
			}

			return printJSON(webhook)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.webhookFlags.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func webhookGetCmd() *cobra.Command {
	var flags webhookGetFlag

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a particular webhook.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)

			webhook, err := client.GetWebhook(flags.webhookID)
			if err != nil {
				return errors.Wrap(err, "failed to query webhook")
			}
			if webhook == nil {
				return nil
			}

			return printJSON(webhook)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.webhookFlags.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func webhookListCmd() *cobra.Command {
	var flags webhookListFlag

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List created webhooks.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			return executeWebhookListCmd(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.webhookFlags.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func executeWebhookListCmd(flags webhookListFlag) error {
	client := model.NewClient(flags.serverAddress)

	paging := getPaging(flags.pagingFlags)
	webhooks, err := client.GetWebhooks(&model.GetWebhooksRequest{
		OwnerID: flags.owner,
		Paging:  paging,
	})
	if err != nil {
		return errors.Wrap(err, "failed to query webhooks")
	}

	if flags.outputToTable {
		table := tablewriter.NewWriter(os.Stdout)
		table.SetAlignment(tablewriter.ALIGN_LEFT)
		table.SetHeader([]string{"ID", "OWNER", "URL"})

		for _, webhook := range webhooks {
			table.Append([]string{webhook.ID, webhook.OwnerID, webhook.URL})
		}
		table.Render()

		return nil
	}

	return printJSON(webhooks)
}

func webhookDeleteCmd() *cobra.Command {
	var flags webhookDeleteFlag

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a webhook.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			if err := client.DeleteWebhook(flags.webhookID); err != nil {
				return errors.Wrap(err, "failed to delete webhook")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.webhookFlags.addFlags(cmd)
			return
		},
	}

	flags.addFlags(cmd)

	return cmd
}
