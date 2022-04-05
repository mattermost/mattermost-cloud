// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gosuri/uilive"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/mattermost/mattermost-cloud/model"
)

func init() {
	dashboardCmd.PersistentFlags().String("server", defaultLocalServerAPI, "The provisioning server whose API will be queried.")

	dashboardCmd.PersistentFlags().Int("refresh-seconds", 10, "The amount of seconds before the dashboard is refreshed with new data.")
}

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "View an auto-refreshing dashboard of all cloud server resources.",
	RunE: func(command *cobra.Command, args []string) error {
		command.SilenceUsage = true

		serverAddress, _ := command.Flags().GetString("server")
		client := model.NewClient(serverAddress)

		refreshSeconds, _ := command.Flags().GetInt("refresh-seconds")
		if refreshSeconds < 1 {
			return errors.Errorf("refresh seconds (%d) must be set to 1 or higher", refreshSeconds)
		}

		writer := uilive.New()
		writer.Start()

		for {
			tableString := &strings.Builder{}
			table := tablewriter.NewWriter(tableString)
			table.SetAlignment(tablewriter.ALIGN_LEFT)
			table.SetHeader([]string{"TYPE", "TOTAL", "STABLE", "WIP"})

			var unstableList []string

			// Clusters
			clusters, err := client.GetClusters(&model.GetClustersRequest{
				Paging: model.AllPagesNotDeleted(),
			})
			if err != nil {
				return errors.Wrap(err, "failed to query clusters")
			}

			clusterCount := len(clusters)
			var clusterStableCount int
			for _, cluster := range clusters {
				if cluster.State == model.ClusterStateStable {
					clusterStableCount++
				} else {
					unstableList = append(unstableList, fmt.Sprintf("Cluster: %s (%s)", cluster.ID, cluster.State))
				}
			}

			table.Append([]string{
				"Cluster",
				toStr(clusterCount),
				toStr(clusterStableCount),
				toStr(clusterCount - clusterStableCount),
			})

			// Installations
			installations, err := client.GetInstallations(&model.GetInstallationsRequest{
				Paging: model.AllPagesNotDeleted(),
			})
			if err != nil {
				return errors.Wrap(err, "failed to query installations")
			}

			installationCount := len(installations)
			var installationStableCount, installationsHibernatingCount int
			for _, installation := range installations {
				switch installation.State {
				case model.ClusterInstallationStateStable:
					installationStableCount++
				case model.InstallationStateHibernating:
					installationsHibernatingCount++
				default:
					unstableList = append(unstableList, fmt.Sprintf("Installation: %s | %s (%s)", installation.ID, installation.DNS, installation.State))
				}
			}

			table.Append([]string{
				"Installation",
				toStr(installationCount),
				fmt.Sprintf("%d (H=%d)", installationStableCount+installationsHibernatingCount, installationsHibernatingCount),
				toStr(installationCount - (installationStableCount + installationsHibernatingCount)),
			})

			clusterInstallations, err := client.GetClusterInstallations(&model.GetClusterInstallationsRequest{
				Paging: model.AllPagesNotDeleted(),
			})
			if err != nil {
				return errors.Wrap(err, "failed to query clusters")
			}

			// Cluster Installations
			clusterInstallationCount := len(clusterInstallations)
			var clusterInstallationStableCount int
			for _, clusterInstallation := range clusterInstallations {
				if clusterInstallation.State == model.ClusterInstallationStateStable {
					clusterInstallationStableCount++
				} else {
					unstableList = append(unstableList, fmt.Sprintf("Cluster Installation: %s (%s)", clusterInstallation.ID, clusterInstallation.State))
				}
			}

			table.Append([]string{
				"Cluster Installation",
				toStr(clusterInstallationCount),
				toStr(clusterInstallationStableCount),
				toStr(clusterInstallationCount - clusterInstallationStableCount),
			})

			table.Render()
			renderedDashboard := fmt.Sprintf("\n### CLOUD DASHBOARD\n\n")
			renderedDashboard += tableString.String()
			for _, entry := range unstableList {
				renderedDashboard += fmt.Sprintf("%s\n", entry)
			}
			if len(unstableList) != 0 {
				renderedDashboard += "\n"
			}

			for i := refreshSeconds; i > 0; i-- {
				fmt.Fprintf(writer, "%s\nUpdating in %d seconds...\n", renderedDashboard, i)
				time.Sleep(time.Second)
			}
		}
	},
}

func toStr(i int) string {
	return strconv.Itoa(i)
}
