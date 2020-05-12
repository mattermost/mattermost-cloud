package main

import (
	"encoding/json"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-cloud/model"
)

func init() {
	clusterCmd.PersistentFlags().String("server", "http://localhost:8075", "The provisioning server whose API will be queried.")

	clusterCreateCmd.Flags().String("provider", "aws", "Cloud provider hosting the cluster.")
	clusterCreateCmd.Flags().String("version", "latest", "The Kubernetes version to target. Use 'latest' or versions such as '1.14.1'.")
	clusterCreateCmd.Flags().String("kops-ami", "", "The AMI to use for the cluster hosts. Leave empty for the default kops image.")
	clusterCreateCmd.Flags().String("size", "SizeAlef500", "The size constant describing the cluster. Add '-HA2' or '-HA3' to the size for multiple master nodes.")
	clusterCreateCmd.Flags().String("zones", "us-east-1a", "The zones where the cluster will be deployed. Use commas to separate multiple zones.")
	clusterCreateCmd.Flags().Bool("allow-installations", true, "Whether the cluster will allow for new installations to be scheduled.")
	clusterCreateCmd.Flags().String("prometheus-version", model.PrometheusDefaultVersion, "The version of Prometheus to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("fluentbit-version", model.FluentbitDefaultVersion, "The version of Fluentbit to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("nginx-version", model.NginxDefaultVersion, "The version of Nginx to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("public-nginx-version", model.PublicNginxDefaultVersion, "The version of Public Nginx to provision. Use 'stable' to provision the latest stable version published upstream.")
	clusterCreateCmd.Flags().String("cert-manager-version", model.CertManagerDefaultVersion, "The version of Cert Manager to provision. Use 'stable' to provision the latest stable version published upstream.")

	clusterProvisionCmd.Flags().String("cluster", "", "The id of the cluster to be provisioned.")
	clusterProvisionCmd.Flags().String("prometheus-version", "", "The version of Prometheus to provision, no change if omitted. Use \"stable\" as an argument to this command to indicate that you wish to remove the pinned version and return the utility to tracking the latest version.")
	clusterProvisionCmd.Flags().String("fluentbit-version", "", "The version of Fluentbit to provision, no change if omitted. Use \"stable\" as an argument to this command to indicate that you wish to remove the pinned version and return the utility to tracking the latest version.")
	clusterProvisionCmd.Flags().String("nginx-version", "", "The version of Nginx to provision, no change if omitted. Use \"stable\" as an argument to this command to indicate that you wish to remove the pinned version and return the utility to tracking the latest version.")
	clusterProvisionCmd.Flags().String("public-nginx-version", "", "The version of Public Nginx to provision, no change if omitted. Use \"stable\" as an argument to this command to indicate that you wish to remove the pinned version and return the utility to tracking the latest version.")
	clusterProvisionCmd.Flags().String("cert-manager-version", "", "The version of Cert Manager to provision, no change if omitted. Use \"stable\" as an argument to this command to indicate that you wish to remove the pinned version and return the utility to tracking the latest version.")
	clusterProvisionCmd.MarkFlagRequired("cluster")

	clusterUpdateCmd.Flags().String("cluster", "", "The id of the cluster to be updated.")
	clusterUpdateCmd.Flags().Bool("allow-installations", true, "Whether the cluster will allow for new installations to be scheduled.")
	clusterUpdateCmd.MarkFlagRequired("cluster")

	clusterUpgradeCmd.Flags().String("cluster", "", "The id of the cluster to be upgraded.")
	clusterUpgradeCmd.Flags().String("version", "latest", "The Kubernetes version to target. Use 'latest' or versions such as '1.14.1'.")
	clusterUpgradeCmd.MarkFlagRequired("cluster")
	clusterUpgradeCmd.MarkFlagRequired("version")

	clusterResizeCmd.Flags().String("cluster", "", "The id of the cluster to be resized.")
	clusterResizeCmd.Flags().String("size", "SizeAlef500", "The size constant describing the cluster.")
	clusterResizeCmd.MarkFlagRequired("cluster")
	clusterResizeCmd.MarkFlagRequired("size")

	clusterDeleteCmd.Flags().String("cluster", "", "The id of the cluster to be deleted.")
	clusterDeleteCmd.MarkFlagRequired("cluster")

	clusterGetCmd.Flags().String("cluster", "", "The id of the cluster to be fetched.")
	clusterGetCmd.MarkFlagRequired("cluster")

	clusterListCmd.Flags().Int("page", 0, "The page of clusters to fetch, starting at 0.")
	clusterListCmd.Flags().Int("per-page", 100, "The number of clusters to fetch per page.")
	clusterListCmd.Flags().Bool("include-deleted", false, "Whether to include deleted clusters.")

	clusterUtilitiesCmd.Flags().String("cluster", "", "The id of the cluster whose utilities are to be fetched.")
	clusterUtilitiesCmd.MarkFlagRequired("cluster")

	clusterCmd.AddCommand(clusterCreateCmd)
	clusterCmd.AddCommand(clusterProvisionCmd)
	clusterCmd.AddCommand(clusterUpdateCmd)
	clusterCmd.AddCommand(clusterUpgradeCmd)
	clusterCmd.AddCommand(clusterResizeCmd)
	clusterCmd.AddCommand(clusterDeleteCmd)
	clusterCmd.AddCommand(clusterGetCmd)
	clusterCmd.AddCommand(clusterListCmd)
	clusterCmd.AddCommand(clusterInstallationCmd)
	clusterCmd.AddCommand(clusterShowStateReport)
	clusterCmd.AddCommand(clusterUtilitiesCmd)
}

var clusterCmd = &cobra.Command{
	Use:   "cluster",
	Short: "Manipulate clusters managed by the provisioning server.",
}

func printJSON(data interface{}) error {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "    ")
	return encoder.Encode(data)
}

var clusterCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		provider, _ := command.Flags().GetString("provider")
		version, _ := command.Flags().GetString("version")
		kopsAMI, _ := command.Flags().GetString("kops-ami")
		size, _ := command.Flags().GetString("size")
		zones, _ := command.Flags().GetString("zones")
		allowInstallations, _ := command.Flags().GetBool("allow-installations")

		cluster, err := client.CreateCluster(&model.CreateClusterRequest{
			Provider:               provider,
			Version:                version,
			KopsAMI:                kopsAMI,
			Size:                   size,
			Zones:                  strings.Split(zones, ","),
			AllowInstallations:     allowInstallations,
			DesiredUtilityVersions: processUtilityFlags(command),
		})
		if err != nil {
			return errors.Wrap(err, "failed to create cluster")
		}

		err = printJSON(cluster)
		if err != nil {
			return err
		}

		return nil
	},
}

var clusterProvisionCmd = &cobra.Command{
	Use:   "provision",
	Short: "Provision/Reprovision a cluster's k8s operators.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)
		clusterID, _ := command.Flags().GetString("cluster")

		var pcr *model.ProvisionClusterRequest = nil
		if desiredUtilityVersions := processUtilityFlags(command); len(desiredUtilityVersions) > 0 {
			pcr = &model.ProvisionClusterRequest{
				DesiredUtilityVersions: desiredUtilityVersions,
			}
		}

		err := client.ProvisionCluster(clusterID, pcr)
		if err != nil {
			return errors.Wrap(err, "failed to provision cluster")
		}

		return nil
	},
}

var clusterUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Updates a cluster's configuration.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterID, _ := command.Flags().GetString("cluster")
		allowInstallations, _ := command.Flags().GetBool("allow-installations")

		cluster, err := client.UpdateCluster(clusterID, &model.UpdateClusterRequest{
			AllowInstallations: allowInstallations,
		})
		if err != nil {
			return errors.Wrap(err, "failed to update cluster")
		}

		err = printJSON(cluster)
		if err != nil {
			return err
		}

		return nil
	},
}

var clusterUpgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade k8s on a cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterID, _ := command.Flags().GetString("cluster")
		version, _ := command.Flags().GetString("version")

		err := client.UpgradeCluster(clusterID, version)
		if err != nil {
			return errors.Wrap(err, "failed to upgrade cluster")
		}

		return nil
	},
}

var clusterResizeCmd = &cobra.Command{
	Use:   "resize",
	Short: "Resize a k8s cluster",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterID, _ := command.Flags().GetString("cluster")
		size, _ := command.Flags().GetString("size")

		err := client.ResizeCluster(clusterID, size)
		if err != nil {
			return errors.Wrap(err, "failed to resize cluster")
		}

		return nil
	},
}

var clusterDeleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterID, _ := command.Flags().GetString("cluster")

		err := client.DeleteCluster(clusterID)
		if err != nil {
			return errors.Wrap(err, "failed to delete cluster")
		}

		return nil
	},
}

var clusterGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get a particular cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		clusterID, _ := command.Flags().GetString("cluster")
		cluster, err := client.GetCluster(clusterID)
		if err != nil {
			return errors.Wrap(err, "failed to query cluster")
		}
		if cluster == nil {
			return nil
		}

		err = printJSON(cluster)
		if err != nil {
			return err
		}

		return nil
	},
}

var clusterListCmd = &cobra.Command{
	Use:   "list",
	Short: "List created clusters.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		page, _ := command.Flags().GetInt("page")
		perPage, _ := command.Flags().GetInt("per-page")
		includeDeleted, _ := command.Flags().GetBool("include-deleted")
		clusters, err := client.GetClusters(&model.GetClustersRequest{
			Page:           page,
			PerPage:        perPage,
			IncludeDeleted: includeDeleted,
		})
		if err != nil {
			return errors.Wrap(err, "failed to query clusters")
		}

		err = printJSON(clusters)
		if err != nil {
			return err
		}

		return nil
	},
}

var clusterUtilitiesCmd = &cobra.Command{
	Use:   "utilities",
	Short: "Show metadata regarding utility services running in a cluster.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)
		clusterID, err := command.Flags().GetString("cluster")
		if err != nil {
			return err
		}

		metadata, err := client.GetClusterUtilities(clusterID)
		if err != nil {
			return err
		}

		err = printJSON(metadata)
		if err != nil {
			return err
		}

		return nil
	},
}

// TODO:
// Instead of showing the state data from the model of the CLI binary, add a new
// API endpoint to return the server's state model.
var clusterShowStateReport = &cobra.Command{
	Use:   "state-report",
	Short: "Shows information regarding changing cluster state.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		err := printJSON(model.GetClusterRequestStateReport())
		if err != nil {
			return err
		}

		return nil
	},
}

func processUtilityFlags(command *cobra.Command) map[string]string {
	prometheusVersion, _ := command.Flags().GetString("prometheus-version")
	fluentbitVersion, _ := command.Flags().GetString("fluentbit-version")
	nginxVersion, _ := command.Flags().GetString("nginx-version")
	publicNginxVersion, _ := command.Flags().GetString("public-nginx-version")
	certManagerVersion, _ := command.Flags().GetString("cert-manager-version")

	utilityVersions := make(map[string]string)

	if prometheusVersion != "" {
		utilityVersions[model.PrometheusCanonicalName] = prometheusVersion
	}

	if fluentbitVersion != "" {
		utilityVersions[model.FluentbitCanonicalName] = fluentbitVersion
	}

	if nginxVersion != "" {
		utilityVersions[model.NginxCanonicalName] = nginxVersion
	}

	if publicNginxVersion != "" {
		utilityVersions[model.PublicNginxCanonicalName] = publicNginxVersion
	}

	if certManagerVersion != "" {
		utilityVersions[model.CertManagerCanonicalName] = certManagerVersion
	}

	return utilityVersions
}
