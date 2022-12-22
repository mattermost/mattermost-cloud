package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/spf13/cobra"
	"time"
)

type subscriptionCreateFlags struct {
	clusterFlags
	name             string
	url              string
	owner            string
	eventType        string
	failureThreshold time.Duration
}

func (flags *subscriptionCreateFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.name, "name", "", "Name of the subscription.")
	command.Flags().StringVar(&flags.url, "url", "", "URL of the subscription.")
	command.Flags().StringVar(&flags.owner, "owner", "", "OwnerID of the subscription.")
	command.Flags().StringVar(&flags.eventType, "event-type", string(model.ResourceStateChangeEventType), "Event type of the subscription.")
	command.Flags().DurationVar(&flags.failureThreshold, "failure-threshold", 0, "Failure threshold of the subscription.")
	_ = command.MarkFlagRequired("url")
	_ = command.MarkFlagRequired("owner")
	_ = command.MarkFlagRequired("event-type")
}

type subscriptionListFlags struct {
	clusterFlags
	pagingFlags
	tableOptions
	owner     string
	eventType string
}

func (flags *subscriptionListFlags) addFlags(command *cobra.Command) {
	flags.pagingFlags.addFlags(command)
	flags.tableOptions.addFlags(command)
	command.Flags().StringVar(&flags.owner, "owner", "", "OwnerID of the subscription.")
	command.Flags().StringVar(&flags.eventType, "event-type", "", "Event type of the subscription.")
}

type subscriptionGetFlags struct {
	clusterFlags
	subID string
}

func (flags *subscriptionGetFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.subID, "subscription", "", "ID of subscription to get")
	_ = command.MarkFlagRequired("subscription")
}

type subscriptionDeleteFlags struct {
	clusterFlags
	subID string
}

func (flags *subscriptionDeleteFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.subID, "subscription", "", "ID of subscription to delete")
	_ = command.MarkFlagRequired("subscription")
}
