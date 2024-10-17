// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newCmdSubscription() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subscription",
		Short: "Manipulate subscriptions managed by the provisioning server.",
	}

	setClusterFlags(cmd)

	cmd.AddCommand(newCmdSubscriptionCreate())
	cmd.AddCommand(newCmdSubscriptionList())
	cmd.AddCommand(newCmdSubscriptionGet())
	cmd.AddCommand(newCmdSubscriptionDelete())

	return cmd
}

func newCmdSubscriptionCreate() *cobra.Command {
	var flags subscriptionCreateFlags

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Creates subscription.",
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

			request := &model.CreateSubscriptionRequest{
				Name:             flags.name,
				URL:              flags.url,
				OwnerID:          flags.owner,
				EventType:        model.EventType(flags.eventType),
				FailureThreshold: flags.failureThreshold,
				Headers:          headers,
			}

			if flags.dryRun {
				return runDryRun(request)
			}

			backup, err := client.CreateSubscription(request)
			if err != nil {
				return errors.Wrap(err, "failed to create subscription")
			}

			return printJSON(backup)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdSubscriptionList() *cobra.Command {

	var flags subscriptionListFlags

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List subscriptions.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)

			paging := getPaging(flags.pagingFlags)

			request := &model.ListSubscriptionsRequest{
				Paging:    paging,
				Owner:     flags.owner,
				EventType: model.EventType(flags.eventType),
			}

			subscriptions, err := client.ListSubscriptions(request)
			if err != nil {
				return errors.Wrap(err, "failed to get backup")
			}

			if enabled, customCols := getTableOutputOption(flags.tableOptions); enabled {
				var keys []string
				var vals [][]string

				if len(customCols) > 0 {
					data := make([]interface{}, 0, len(subscriptions))
					for _, elem := range subscriptions {
						data = append(data, elem)
					}
					keys, vals, err = prepareTableData(customCols, data)
					if err != nil {
						return errors.Wrap(err, "failed to prepare table output")
					}
				} else {
					keys, vals = defaultSubscriptionsTableData(subscriptions)
				}

				printTable(keys, vals)
				return nil
			}

			return printJSON(subscriptions)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func defaultSubscriptionsTableData(subscriptions []*model.Subscription) ([]string, [][]string) {
	keys := []string{"ID", "EVENT TYPE", "OWNER", "LAST DELIVERY ATTEMPT", "LAST DELIVERY STATUS"}
	vals := make([][]string, 0, len(subscriptions))

	for _, sub := range subscriptions {
		vals = append(vals, []string{
			sub.ID,
			string(sub.EventType),
			sub.OwnerID,
			model.TimeFromMillis(sub.LastDeliveryAttemptAt).Format("2006-01-02 15:04:05 -0700 MST"),
			string(sub.LastDeliveryStatus),
		})
	}

	return keys, vals
}

func newCmdSubscriptionGet() *cobra.Command {
	var flags subscriptionGetFlags

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get subscription.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)

			subscription, err := client.GetSubscription(flags.subID)
			if err != nil {
				return errors.Wrap(err, "failed to get subscription")
			}

			return printJSON(subscription)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func newCmdSubscriptionDelete() *cobra.Command {
	var flags subscriptionDeleteFlags

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete subscription.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := createClient(command.Context(), flags.clusterFlags)
			if err := client.DeleteSubscription(flags.subID); err != nil {
				return errors.Wrap(err, "failed to delete subscription")
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
