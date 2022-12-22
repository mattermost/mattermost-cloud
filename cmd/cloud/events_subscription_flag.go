package main

import (
	"time"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/spf13/cobra"
)

type subscriptionCreateFlags struct {
	clusterFlags
	name             string
	url              string
	owner            string
	eventType        string
	failureThreshold time.Duration
}

func (flags *subscriptionCreateFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.name, "name", "", "Name of the subscription.")
	cmd.Flags().StringVar(&flags.url, "url", "", "URL of the subscription.")
	cmd.Flags().StringVar(&flags.owner, "owner", "", "OwnerID of the subscription.")
	cmd.Flags().StringVar(&flags.eventType, "event-type", string(model.ResourceStateChangeEventType), "Event type of the subscription.")
	cmd.Flags().DurationVar(&flags.failureThreshold, "failure-threshold", 0, "Failure threshold of the subscription.")
	_ = cmd.MarkFlagRequired("url")
	_ = cmd.MarkFlagRequired("owner")
	_ = cmd.MarkFlagRequired("event-type")
}

type subscriptionListFlags struct {
	clusterFlags
	pagingFlags
	tableOptions
	owner     string
	eventType string
}

func (flags *subscriptionListFlags) addFlags(cmd *cobra.Command) {
	flags.pagingFlags.addFlags(cmd)
	flags.tableOptions.addFlags(cmd)
	cmd.Flags().StringVar(&flags.owner, "owner", "", "OwnerID of the subscription.")
	cmd.Flags().StringVar(&flags.eventType, "event-type", "", "Event type of the subscription.")
}

type subscriptionGetFlags struct {
	clusterFlags
	subID string
}

func (flags *subscriptionGetFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.subID, "subscription", "", "ID of subscription to get")
	_ = cmd.MarkFlagRequired("subscription")
}

type subscriptionDeleteFlags struct {
	clusterFlags
	subID string
}

func (flags *subscriptionDeleteFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.subID, "subscription", "", "ID of subscription to delete")
	_ = cmd.MarkFlagRequired("subscription")
}
