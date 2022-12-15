package main

import "github.com/spf13/cobra"

type groupAnnotationAddFlags struct {
	clusterFlags
	groupID     string
	annotations []string
}

func (flags *groupAnnotationAddFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.groupID, "group", "", "The id of the group to be annotated.")
	command.Flags().StringSliceVar(&flags.annotations, "annotations", []string{}, "Additional annotations for the group. Accepts multiple values, for example: '... --annotation abc --annotation def'")

	_ = command.MarkFlagRequired("group")
	_ = command.MarkFlagRequired("annotations")
}

type groupAnnotationDeleteFlags struct {
	clusterFlags
	groupID    string
	annotation string
}

func (flags *groupAnnotationDeleteFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.groupID, "group", "", "The id of the group from which annotations should be removed.")
	command.Flags().StringVar(&flags.annotation, "annotation", "", "Name of the annotation to be removed from the group.")

	_ = command.MarkFlagRequired("group")
	_ = command.MarkFlagRequired("annotation")
}
