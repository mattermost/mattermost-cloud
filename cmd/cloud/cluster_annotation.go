package main

import (
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func init() {
	clusterAnnotationAddCmd.Flags().StringArray("annotation", []string{}, "Additional annotations for the cluster. Accepts multiple values, for example: '... --annotation abc --annotation def'")
	clusterAnnotationAddCmd.Flags().String("cluster", "", "The id of the cluster to be annotated.")
	clusterAnnotationAddCmd.MarkFlagRequired("cluster")
	clusterAnnotationAddCmd.MarkFlagRequired("annotation")

	clusterAnnotationDeleteCmd.Flags().String("annotation", "", "Name of the Annotation to be removed from the Cluster.")
	clusterAnnotationDeleteCmd.Flags().String("cluster", "", "The id of the cluster from which annotation should be removed.")
	clusterAnnotationDeleteCmd.MarkFlagRequired("cluster")
	clusterAnnotationDeleteCmd.MarkFlagRequired("annotation")

	clusterAnnotationCmd.AddCommand(clusterAnnotationAddCmd)
	clusterAnnotationCmd.AddCommand(clusterAnnotationDeleteCmd)
}

var clusterAnnotationCmd = &cobra.Command{
	Use:   "annotation",
	Short: "Manipulate annotations of clusters managed by the provisioning server.",
}

var clusterAnnotationAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Adds Annotations to the Cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterID, _ := command.Flags().GetString("cluster")
		annotations, _ := command.Flags().GetStringArray("annotation")

		request := newAddAnnotationsRequest(annotations)

		dryRun, _ := command.Flags().GetBool("dry-run")
		if dryRun {
			return runDryRun(request)
		}

		cluster, err := client.AddClusterAnnotations(clusterID, request)
		if err != nil {
			return errors.Wrap(err, "failed to add cluster annotations")
		}

		err = printJSON(cluster)
		if err != nil {
			return errors.Wrap(err, "failed to print cluster annotations response")
		}

		return nil
	},
}

var clusterAnnotationDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Deletes Annotation from the Cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterID, _ := command.Flags().GetString("cluster")
		annotation, _ := command.Flags().GetString("annotation")

		err := client.DeleteClusterAnnotation(clusterID, annotation)
		if err != nil {
			return errors.Wrap(err, "failed to delete cluster annotations")
		}

		return nil
	},
}

func newAddAnnotationsRequest(annotations []string) *model.AddAnnotationsRequest {
	return &model.AddAnnotationsRequest{
		Annotations: annotations,
	}
}

func runDryRun(request interface{}) error {
	err := printJSON(request)
	if err != nil {
		return errors.Wrap(err, "failed to print API request")
	}

	return nil
}
