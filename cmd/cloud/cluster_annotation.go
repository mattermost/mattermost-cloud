// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newCmdClusterAnnotation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "annotation",
		Short: "Manipulate annotations of clusters managed by the provisioning server.",
	}

	cmd.AddCommand(newCmdClusterAnnotationAdd())
	cmd.AddCommand(newCmdClusterAnnotationDelete())

	return cmd
}

func newCmdClusterAnnotationAdd() *cobra.Command {
	var flags clusterAnnotationAddFlags

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Adds annotations to the cluster.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			return executeClusterAnnotationAddCmd(flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterAnnotationAddCmd(flags clusterAnnotationAddFlags) error {

	client := model.NewClient(flags.serverAddress)

	request := newAddAnnotationsRequest(flags.annotations)

	if flags.dryRun {
		return runDryRun(request)
	}

	cluster, err := client.AddClusterAnnotations(flags.cluster, request)
	if err != nil {
		return errors.Wrap(err, "failed to add cluster annotations")
	}

	if err = printJSON(cluster); err != nil {
		return errors.Wrap(err, "failed to print cluster annotations response")
	}

	return nil
}

func newCmdClusterAnnotationDelete() *cobra.Command {
	var flags clusterAnnotationDeleteFlags

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes Annotation from the Cluster.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.SilenceUsage = true

			client := model.NewClient(flags.serverAddress)

			if err := client.DeleteClusterAnnotation(flags.cluster, flags.annotation); err != nil {
				return errors.Wrap(err, "failed to delete cluster annotations")
			}
			return nil
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
			return
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func newAddAnnotationsRequest(annotations []string) *model.AddAnnotationsRequest {
	return &model.AddAnnotationsRequest{
		Annotations: annotations,
	}
}
