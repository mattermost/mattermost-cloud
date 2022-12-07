package main

import "github.com/spf13/cobra"

type clusterAnnotationAddFlags struct {
	clusterFlags
	cluster     string
	annotations []string
}

func (flags *clusterAnnotationAddFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.cluster, "cluster", "", "The id of the cluster to be annotated.")
	command.Flags().StringArrayVar(&flags.annotations, "annotation", []string{}, "Additional annotations for the cluster. Accepts multiple values, for example: '... --annotation abc --annotation def'")
	_ = command.MarkFlagRequired("cluster")
	_ = command.MarkFlagRequired("annotation")
}

type clusterAnnotationDeleteFlags struct {
	clusterFlags
	cluster    string
	annotation string
}

func (flags *clusterAnnotationDeleteFlags) addFlags(command *cobra.Command) {
	command.Flags().StringVar(&flags.cluster, "cluster", "", "The id of the cluster from which annotation should be removed.")
	command.Flags().StringVar(&flags.annotation, "annotation", "", "Name of the annotation to be removed from the cluster.")
	_ = command.MarkFlagRequired("cluster")
	_ = command.MarkFlagRequired("annotation")
}
