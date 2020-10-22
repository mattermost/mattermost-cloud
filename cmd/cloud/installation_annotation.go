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
	installationAnnotationAddCmd.Flags().StringArray("annotation", []string{}, "Additional annotations for the installation. Accepts multiple values, for example: '... --annotation abc --annotation def'")
	installationAnnotationAddCmd.Flags().String("installation", "", "The id of the installation to be annotated.")
	installationAnnotationAddCmd.MarkFlagRequired("installation")
	installationAnnotationAddCmd.MarkFlagRequired("annotation")

	installationAnnotationDeleteCmd.Flags().String("annotation", "", "Name of the Annotation to be removed from the Installation.")
	installationAnnotationDeleteCmd.Flags().String("installation", "", "The id of the installation from which annotations should be removed.")
	installationAnnotationDeleteCmd.MarkFlagRequired("installation")
	installationAnnotationDeleteCmd.MarkFlagRequired("annotation")

	installationAnnotationCmd.AddCommand(installationAnnotationAddCmd)
	installationAnnotationCmd.AddCommand(installationAnnotationDeleteCmd)
}

var installationAnnotationCmd = &cobra.Command{
	Use:   "annotation",
	Short: "Manipulate annotations of installations managed by the provisioning server.",
}

var installationAnnotationAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Adds Annotations to the Installation.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		annotations, _ := command.Flags().GetStringArray("annotation")

		request := newAddAnnotationsRequest(annotations)

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			return runDryRun(request)
		}

		cluster, err := client.AddInstallationAnnotations(installationID, request)
		if err != nil {
			return errors.Wrap(err, "failed to add installation annotations")
		}

		err = printJSON(cluster)
		if err != nil {
			return errors.Wrap(err, "failed to print installation annotations response")
		}

		return nil
	},
}

var installationAnnotationDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Deletes Annotation from the Installation.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		installationID, _ := command.Flags().GetString("installation")
		annotation, _ := command.Flags().GetString("annotation")

		err := client.DeleteInstallationAnnotation(installationID, annotation)
		if err != nil {
			return errors.Wrap(err, "failed to delete installation annotations")
		}

		return nil
	},
}
