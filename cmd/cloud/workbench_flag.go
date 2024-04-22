package main

import "github.com/spf13/cobra"

func setWorkbenchFlags(command *cobra.Command) {
	command.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
	command.PersistentFlags().String("state-store", "dev.cloud.mattermost.com", "The S3 bucket used to store cluster state.")
}

type workbenchFlags struct {
	clusterFlags
	s3StateStore string
}

func (flags *workbenchFlags) addFlags(command *cobra.Command) {
	flags.serverAddress, _ = command.Flags().GetString("server")
	flags.s3StateStore, _ = command.Flags().GetString("state-store")
}

type workbenchClusterFlag struct {
	workbenchFlags
	clusterID string
}

func (flags *workbenchClusterFlag) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.clusterID, "cluster", "", "The id of the cluster to work on.")
	_ = command.MarkFlagRequired("cluster")
}
