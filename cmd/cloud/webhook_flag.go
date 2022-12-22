package main

import "github.com/spf13/cobra"

func setWebhookFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
}

type webhookFlags struct {
	serverAddress string
}

func (flags *webhookFlags) addFlags(cmd *cobra.Command) {
	flags.serverAddress, _ = cmd.Flags().GetString("server")
}

type webhookCreateFlag struct {
	webhookFlags
	ownerID string
	url     string
}

func (flags *webhookCreateFlag) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.ownerID, "owner", "", "An opaque identifier describing the owner of the webhook.")
	cmd.Flags().StringVar(&flags.url, "url", "", "The callback URL of the webhook.")
	_ = cmd.MarkFlagRequired("owner")
	_ = cmd.MarkFlagRequired("url")
}

type webhookGetFlag struct {
	webhookFlags
	webhookID string
}

func (flags *webhookGetFlag) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.webhookID, "webhook", "", "The id of the webhook to be fetched.")
	_ = cmd.MarkFlagRequired("webhook")
}

type webhookListFlag struct {
	webhookFlags
	pagingFlags
	owner         string
	outputToTable bool
}

func (flags *webhookListFlag) addFlags(cmd *cobra.Command) {
	flags.pagingFlags.addFlags(cmd)
	cmd.Flags().StringVar(&flags.owner, "owner", "", "The owner by which to filter webhooks.")
	cmd.Flags().BoolVar(&flags.outputToTable, "table", false, "Whether to display the returned webhook list in a table or not")
}

type webhookDeleteFlag struct {
	webhookFlags
	webhookID string
}

func (flags *webhookDeleteFlag) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.webhookID, "webhook", "", "The id of the webhook to be deleted.")
	_ = cmd.MarkFlagRequired("webhook")
}
