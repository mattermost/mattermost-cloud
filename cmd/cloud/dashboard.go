// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/gosuri/uilive"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/olekukonko/tablewriter"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func newCmdDashboard() *cobra.Command {

	var flags dashboardFlags

	cmd := &cobra.Command{
		Use:   "dashboard",
		Short: "View an auto-refreshing dashboard of all cloud server resources.",
		RunE: func(command *cobra.Command, args []string) error {
			command.SilenceUsage = true
			return executeDashboardCmd(command.Context(), flags)
		},
	}

	flags.addFlags(cmd)

	return cmd
}

func executeDashboardCmd(ctx context.Context, flags dashboardFlags) error {
	client := createClient(ctx, flags.clusterFlags)
	if flags.refreshSeconds < 1 {
		return errors.Errorf("refresh seconds (%d) must be set to 1 or higher", flags.refreshSeconds)
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
		start := time.Now()
		clusters, err := client.GetClusters(&model.GetClustersRequest{
			Paging: model.AllPagesNotDeleted(),
		})
		if err != nil {
			return errors.Wrap(err, "failed to query clusters")
		}
		clusterQueryTime := time.Since(start)

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
		start = time.Now()
		installations, err := client.GetInstallations(&model.GetInstallationsRequest{
			Paging: model.AllPagesNotDeleted(),
		})
		if err != nil {
			return errors.Wrap(err, "failed to query installations")
		}
		installationQueryTime := time.Since(start)

		installationCount := len(installations)
		var installationStableCount, installationsHibernatingCount, installationsPendingDeletionCount int
		for _, installation := range installations {
			switch installation.State {
			case model.ClusterInstallationStateStable:
				installationStableCount++
			case model.InstallationStateHibernating:
				installationsHibernatingCount++
			case model.InstallationStateDeletionPending:
				installationsPendingDeletionCount++
			default:
				var domainName string
				if len(installation.DNSRecords) > 0 {
					domainName = installation.DNSRecords[0].DomainName
				}
				unstableList = append(unstableList, fmt.Sprintf("Installation: %s | %s (%s)", installation.ID, domainName, installation.State))
			}
		}

		table.Append([]string{
			"Installation",
			toStr(installationCount),
			fmt.Sprintf("%d (H=%d, DP=%d)", installationStableCount+installationsHibernatingCount, installationsHibernatingCount, installationsPendingDeletionCount),
			toStr(installationCount - (installationStableCount + installationsHibernatingCount + installationsPendingDeletionCount)),
		})

		// Cluster Installations
		start = time.Now()
		clusterInstallations, err := client.GetClusterInstallations(&model.GetClusterInstallationsRequest{
			Paging: model.AllPagesNotDeleted(),
		})
		if err != nil {
			return errors.Wrap(err, "failed to query clusters")
		}
		ciQueryTime := time.Since(start)

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
		renderedDashboard := "\n### CLOUD DASHBOARD\n"
		renderedDashboard += fmt.Sprintf("[ Query Time Stats: CLSR=%s, INST=%s, CLIN=%s ]\n\n",
			clusterQueryTime.Round(time.Millisecond).String(),
			installationQueryTime.Round(time.Millisecond).String(),
			ciQueryTime.Round(time.Millisecond).String())
		renderedDashboard += tableString.String()
		for _, entry := range unstableList {
			renderedDashboard += fmt.Sprintf("%s\n", entry)
		}
		if len(unstableList) != 0 {
			renderedDashboard += "\n"
		}

		for i := flags.refreshSeconds; i > 0; i-- {
			_, _ = fmt.Fprintf(writer, "%s\nUpdating in %d seconds...\n", renderedDashboard, i)
			time.Sleep(time.Second)
		}
	}
}

func toStr(i int) string {
	return strconv.Itoa(i)
}
