// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"context"

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
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			return executeClusterAnnotationAddCmd(command.Context(), flags)
		},
		PreRun: func(cmd *cobra.Command, args []string) {
			flags.clusterFlags.addFlags(cmd)
		},
	}
	flags.addFlags(cmd)

	return cmd
}

func executeClusterAnnotationAddCmd(ctx context.Context, flags clusterAnnotationAddFlags) error {

	client := createClient(ctx, flags.clusterFlags)

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
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := createClient(command.Context(), flags.clusterFlags)

			if err := client.DeleteClusterAnnotation(flags.cluster, flags.annotation); err != nil {
				return errors.Wrap(err, "failed to delete cluster annotations")
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

func newAddAnnotationsRequest(annotations []string) *model.AddAnnotationsRequest {
	return &model.AddAnnotationsRequest{
		Annotations: annotations,
	}
}
