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
	subscriptionCmd.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
	subscriptionCmd.PersistentFlags().Bool("dry-run", false, "When set to true, only print the API request without sending it.")

	subscriptionCreateCmd.Flags().String("name", "", "Name of the subscription.")
	subscriptionCreateCmd.Flags().String("url", "", "URL of the subscription.")
	subscriptionCreateCmd.Flags().String("owner", "", "OwnerID of the subscription.")
	subscriptionCreateCmd.Flags().String("event-type", string(model.ResourceStateChangeEventType), "Event type of the subscription.")
	subscriptionCreateCmd.Flags().String("failure-threshold", "", "Failure threshold of the subscription.")
	subscriptionCreateCmd.MarkFlagRequired("url")
	subscriptionCreateCmd.MarkFlagRequired("owner")
	subscriptionCreateCmd.MarkFlagRequired("event-type")

	subscriptionListCmd.Flags().String("owner", "", "OwnerID of the subscription.")
	subscriptionListCmd.Flags().String("event-type", "", "Event type of the subscription.")
	registerPagingFlags(subscriptionListCmd)
	registerTableOutputFlags(subscriptionListCmd)

	subscriptionGetCmd.Flags().String("subscription", "", "ID of subscription to get")
	subscriptionGetCmd.MarkFlagRequired("subscription")

	subscriptionDeleteCmd.Flags().String("subscription", "", "ID of subscription to delete")
	subscriptionDeleteCmd.MarkFlagRequired("subscription")

	subscriptionCmd.AddCommand(subscriptionCreateCmd)
	subscriptionCmd.AddCommand(subscriptionListCmd)
	subscriptionCmd.AddCommand(subscriptionGetCmd)
	subscriptionCmd.AddCommand(subscriptionDeleteCmd)
}

var subscriptionCmd = &cobra.Command{
	Use:   "subscription",
	Short: "Manipulate subscriptions managed by the provisioning server.",
}

var subscriptionCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Creates subscription.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		name, _ := command.Flags().GetString("name")
		url, _ := command.Flags().GetString("url")
		owner, _ := command.Flags().GetString("owner")
		eventType, _ := command.Flags().GetString("event-type")
		failureThreshold, _ := command.Flags().GetDuration("failure-threshold")

		request := &model.CreateSubscriptionRequest{
			Name:             name,
			URL:              url,
			OwnerID:          owner,
			EventType:        model.EventType(eventType),
			FailureThreshold: failureThreshold,
		}

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			err := printJSON(request)
			if err != nil {
				return errors.Wrap(err, "failed to print API request")
			}

			return nil
		}

		backup, err := client.CreateSubscription(request)
		if err != nil {
			return errors.Wrap(err, "failed to create subscription")
		}

		return printJSON(backup)
	},
}

var subscriptionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List subscriptions.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		owner, _ := command.Flags().GetString("owner")
		eventType, _ := command.Flags().GetString("event-type")
		paging := parsePagingFlags(command)

		request := &model.ListSubscriptionsRequest{
			Paging:    paging,
			Owner:     owner,
			EventType: model.EventType(eventType),
		}

		subscriptions, err := client.ListSubscriptions(request)
		if err != nil {
			return errors.Wrap(err, "failed to get backup")
		}

		if enabled, customCols := tableOutputEnabled(command); enabled {
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

		err = printJSON(subscriptions)
		if err != nil {
			return err
		}

		return nil
	},
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

var subscriptionGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get subscription.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		subID, _ := command.Flags().GetString("subscription")

		subscription, err := client.GetSubscription(subID)
		if err != nil {
			return errors.Wrap(err, "failed to get subscription")
		}

		err = printJSON(subscription)
		if err != nil {
			return err
		}

		return nil
	},
}

var subscriptionDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete subscription.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		subID, _ := command.Flags().GetString("subscription")

		err := client.DeleteSubscription(subID)
		if err != nil {
			return errors.Wrap(err, "failed to delete subscription")
		}

		return nil
	},
}
