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

func (flags *installationAnnotationAddFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation to be annotated.")
	cmd.Flags().StringArrayVar(&flags.annotations, "annotation", []string{}, "Additional annotations for the installation. Accepts multiple values, for example: '... --annotation abc --annotation def'")

	_ = cmd.MarkFlagRequired("installation")
	_ = cmd.MarkFlagRequired("annotation")
}

type installationAnnotationDeleteFlags struct {
	clusterFlags
	installationID string
	annotation     string
}

func (flags *installationAnnotationDeleteFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.installationID, "installation", "", "The id of the installation from which annotations should be removed.")
	cmd.Flags().StringVar(&flags.annotation, "annotation", "", "Name of the annotation to be removed from the installation.")

	_ = cmd.MarkFlagRequired("installation")
	_ = cmd.MarkFlagRequired("annotation")
}
