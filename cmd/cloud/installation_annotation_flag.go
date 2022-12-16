// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import "github.com/spf13/cobra"

type installationAnnotationAddFlags struct {
	clusterFlags
	installationID string
	annotations    []string
}

func (flags *installationAnnotationAddFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to be annotated.")
	command.Flags().StringArrayVar(&flags.annotations, "annotation", []string{}, "Additional annotations for the installation. Accepts multiple values, for example: '... --annotation abc --annotation def'")

	_ = command.MarkFlagRequired("installation")
	_ = command.MarkFlagRequired("annotation")
}

type installationAnnotationDeleteFlags struct {
	clusterFlags
	installationID string
	annotation     string
}

func (flags *installationAnnotationDeleteFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation from which annotations should be removed.")
	command.Flags().StringVar(&flags.annotation, "annotation", "", "Name of the annotation to be removed from the installation.")

	_ = command.MarkFlagRequired("installation")
	_ = command.MarkFlagRequired("annotation")
}
