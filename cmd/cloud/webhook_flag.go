// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/spf13/cobra"
)

func setWebhookFlags(command *cobra.Command) {
	command.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
}

type webhookFlags struct {
	serverAddress string
}

func (flags *webhookFlags) addFlags(command *cobra.Command) {
	flags.serverAddress, _ = command.Flags().GetString("server")
}

type webhookCreateFlag struct {
	webhookFlags
	ownerID        string
	url            string
	headers        map[string]string
	headersFromEnv map[string]string
}

func (flags *webhookCreateFlag) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.ownerID, "owner", "", "An opaque identifier describing the owner of the webhook.")
	command.Flags().StringVar(&flags.url, "url", "", "The callback URL of the webhook.")
	command.Flags().StringToStringVar(&flags.headers, "header", nil, "a header that should be sent with the request")
	command.Flags().StringToStringVar(&flags.headersFromEnv, "header-from-env", nil, "a header that should be sent with the request, with values read from environment variables")
	_ = command.MarkFlagRequired("owner")
	_ = command.MarkFlagRequired("url")
}

type webhookGetFlag struct {
	webhookFlags
	webhookID string
}

func (flags *webhookGetFlag) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.webhookID, "webhook", "", "The id of the webhook to be fetched.")
	_ = command.MarkFlagRequired("webhook")
}

type webhookListFlag struct {
	webhookFlags
	pagingFlags
	owner         string
	outputToTable bool
}

func (flags *webhookListFlag) addFlags(command *cobra.Command) {
	flags.pagingFlags.addFlags(command)
	command.Flags().StringVar(&flags.owner, "owner", "", "The owner by which to filter webhooks.")
	command.Flags().BoolVar(&flags.outputToTable, "table", false, "Whether to display the returned webhook list in a table or not")
}

type webhookDeleteFlag struct {
	webhookFlags
	webhookID string
}

func (flags *webhookDeleteFlag) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.webhookID, "webhook", "", "The id of the webhook to be deleted.")
	_ = command.MarkFlagRequired("webhook")
}
