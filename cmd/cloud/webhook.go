// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"context"
	"fmt"
	"os"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newCmdWebhook() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "webhook",
		Short: "Manipulate webhooks managed by the provisioning server.",
	}

	setWebhookFlags(cmd)

	cmd.AddCommand(newCmdWebhookCreate())
	cmd.AddCommand(newCmdWebhookGet())
	cmd.AddCommand(newCmdWebhookList())
	cmd.AddCommand(newCmdWebhookDelete())

	return cmd
}

func newCmdWebhookCreate() *cobra.Command {
	var flags webhookCreateFlag

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a webhook.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)

			var headers model.Headers
			for key, value := range flags.headers {
				valueInner := value
				headers = append(headers, model.WebhookHeader{
					Key:   key,
					Value: &valueInner,
				})
			}

			for key, value := range flags.headersFromEnv {
				valueInner := value
				headers = append(headers, model.WebhookHeader{
					Key:          key,
					ValueFromEnv: &valueInner,
				})
			}

			if err := headers.Validate(); err != nil {
				return errors.Wrap(err, "failed to validate webhook headers")
			}

			webhook, err := client.CreateWebhook(&model.CreateWebhookRequest{
				OwnerID: flags.ownerID,
				URL:     flags.url,
				Headers: headers,
			})
			if err != nil {
				return errors.Wrap(err, "failed to create webhook")
			}

			return printJSON(webhook)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.webhookFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdWebhookGet() *cobra.Command {
	var flags webhookGetFlag

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get a particular webhook.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)

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
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdWebhookList() *cobra.Command {
	var flags webhookListFlag

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List created webhooks.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeWebhookListCmd(command.Context(), flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.webhookFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func executeWebhookListCmd(ctx context.Context, flags webhookListFlag) error {
	client := createClient(ctx, flags.clusterFlags)

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
		table.SetHeader([]string{"ID", "OWNER", "URL", "HTTP HEADERS"})

		for _, webhook := range webhooks {
			table.Append([]string{webhook.ID, webhook.OwnerID, webhook.URL, fmt.Sprintf("%d", webhook.Headers.Count())})
		}
		table.Render()

		return nil
	}

	return printJSON(webhooks)
}

func newCmdWebhookDelete() *cobra.Command {
	var flags webhookDeleteFlag

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a webhook.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)
			if err := client.DeleteWebhook(flags.webhookID); err != nil {
				return errors.Wrap(err, "failed to delete webhook")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.webhookFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}
