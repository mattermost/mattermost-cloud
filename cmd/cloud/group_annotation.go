// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newCmdGroupAnnotation() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "annotation",
		Short: "Manipulate annotations of group managed by the provisioning server.",
	}

	cmd.AddCommand(newCmdGroupAnnotationAdd())
	cmd.AddCommand(newCmdGroupAnnotationDelete())

	return cmd
}

func newCmdGroupAnnotationAdd() *cobra.Command {

	var flags groupAnnotationAddFlags

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Adds annotations to the group.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			request := newAddAnnotationsRequest(flags.annotations)
			if flags.dryRun {
				return runDryRun(request)
			}

			group, err := client.AddGroupAnnotations(flags.groupID, request)
			if err != nil {
				return errors.Wrap(err, "failed to add group annotations")
			}

			if err = printJSON(group); err != nil {
				return errors.Wrap(err, "failed to print group annotations response")
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

func newCmdGroupAnnotationDelete() *cobra.Command {
	var flags groupAnnotationDeleteFlags

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Deletes Annotation from the group.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			client := model.NewClient(flags.serverAddress)
			if err := client.DeleteGroupAnnotation(flags.groupID, flags.annotation); err != nil {
				return errors.Wrap(err, "failed to delete group annotations")
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
