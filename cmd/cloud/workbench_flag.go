package main

import "github.com/spf13/cobra"

func setWorkbenchFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")
	cmd.PersistentFlags().String("state-store", "dev.cloud.mattermost.com", "The S3 bucket used to store cluster state.")
}

type workbenchFlags struct {
	serverAddress string
	s3StateStore  string
}

func (flags *workbenchFlags) addFlags(cmd *cobra.Command) {
	flags.serverAddress, _ = cmd.Flags().GetString("server")
	flags.s3StateStore, _ = cmd.Flags().GetString("state-store")
}

type workbenchClusterFlag struct {
	workbenchFlags
	clusterID string
}

func (flags *workbenchClusterFlag) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.clusterID, "cluster", "", "The id of the cluster to work on.")
	_ = cmd.MarkFlagRequired("cluster")
}
