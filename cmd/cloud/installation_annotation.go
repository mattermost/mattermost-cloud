// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func installationAnnotationCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "annotation",
		Short: "Manipulate annotations of installations managed by the provisioning server.",
	}

	cmd.AddCommand(installationAnnotationAddCmd())
	cmd.AddCommand(installationAnnotationDeleteCmd())

	return cmd
}

func installationAnnotationAddCmd() *cobra.Command {
	var flags installationAnnotationAddFlags

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Adds annotations to the installation.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := model.NewClient(flags.serverAddress)

			request := newAddAnnotationsRequest(flags.annotations)

			if flags.dryRun {
				return runDryRun(request)
			}

			cluster, err := client.AddInstallationAnnotations(flags.installationID, request)
			if err != nil {
				return errors.Wrap(err, "failed to add installation annotations")
			}

			if err = printJSON(cluster); err != nil {
				return errors.Wrap(err, "failed to print installation annotations response")
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

func installationAnnotationDeleteCmd() *cobra.Command {
	var flags installationAnnotationDeleteFlags

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes Annotation from the Installation.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true

			client := model.NewClient(flags.serverAddress)

			if err := client.DeleteInstallationAnnotation(flags.installationID, flags.annotation); err != nil {
				return errors.Wrap(err, "failed to delete installation annotations")
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
