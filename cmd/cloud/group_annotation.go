// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	groupAnnotationAddCmd.Flags().StringArray("annotation", []string{}, "Additional annotations for the group. Accepts multiple values, for example: '... --annotation abc --annotation def'")
	groupAnnotationAddCmd.Flags().String("group", "", "The id of the group to be annotated.")
	groupAnnotationAddCmd.MarkFlagRequired("group")
	groupAnnotationAddCmd.MarkFlagRequired("annotaoion")

	groupAnnotationDeleteCmd.Flags().String("annotation", "", "Name of the annotation to be removed from the group.")
	groupAnnotationDeleteCmd.Flags().String("group", "", "The id of the group from which annotations should be removed.")
	groupAnnotationDeleteCmd.MarkFlagRequired("group")
	groupAnnotationDeleteCmd.MarkFlagRequired("annotation")

	groupAnnotationCmd.AddCommand(groupAnnotationAddCmd)
	groupAnnotationCmd.AddCommand(groupAnnotationDeleteCmd)
}

var groupAnnotationCmd = &cobra.Command{
	Use:   "annotation",
	Short: "Manipulate annotations of group managed by the provisioning server.",
}

var groupAnnotationAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Adds annotations to the group.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		groupID, _ := command.Flags().GetString("group")
		annotations, _ := command.Flags().GetStringArray("annotation")

		request := newAddAnnotationsRequest(annotations)

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			return runDryRun(request)
		}

		group, err := client.AddGroupAnnotations(groupID, request)
		if err != nil {
			return errors.Wrap(err, "failed to add group annotations")
		}

		err = printJSON(group)
		if err != nil {
			return errors.Wrap(err, "failed to print group annotations response")
		}

		return nil
	},
}

var groupAnnotationDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Deletes Annotation from the group.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		groupID, _ := command.Flags().GetString("group")
		annotation, _ := command.Flags().GetString("annotation")

		err := client.DeleteGroupAnnotation(groupID, annotation)
		if err != nil {
			return errors.Wrap(err, "failed to delete group annotations")
		}

		return nil
	},
}
