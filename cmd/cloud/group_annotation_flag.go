package main

import "github.com/spf13/cobra"

type groupAnnotationAddFlags struct {
	clusterFlags
	groupID     string
	annotations []string
}

func (flags *groupAnnotationAddFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.groupID, "group", "", "The id of the group to be annotated.")
	cmd.Flags().StringArrayVar(&flags.annotations, "annotation", []string{}, "Additional annotations for the group. Accepts multiple values, for example: '... --annotation abc --annotation def'")

	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("annotation")
}

type groupAnnotationDeleteFlags struct {
	clusterFlags
	groupID    string
	annotation string
}

func (flags *groupAnnotationDeleteFlags) addFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&flags.groupID, "group", "", "The id of the group from which annotations should be removed.")
	cmd.Flags().StringVar(&flags.annotation, "annotation", "", "Name of the annotation to be removed from the group.")

	_ = cmd.MarkFlagRequired("group")
	_ = cmd.MarkFlagRequired("annotation")
}
