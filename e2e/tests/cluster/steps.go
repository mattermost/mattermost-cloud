// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//+build e2e

package cluster

import (
	"github.com/mattermost/mattermost-cloud/e2e/workflow"
)

func clusterLifecycleSteps(clusterSuite *workflow.ClusterSuite, installationSuite *workflow.InstallationSuite) []*workflow.Step {
	return []*workflow.Step{
		{
			Name:      "CreateCluster",
			Func:      clusterSuite.CreateCluster,
			DependsOn: []string{},
		},
		{
			Name:      "CreateInstallation",
			Func:      installationSuite.CreateInstallation,
			DependsOn: []string{"CreateCluster"},
		},
		{
			Name:      "GetCI",
			Func:      installationSuite.GetCI,
			DependsOn: []string{"CreateInstallation"},
		},
		{
			Name:      "PopulateSampleData",
			Func:      installationSuite.PopulateSampleData,
			DependsOn: []string{"GetCI"},
		},
		{
			Name:      "ProvisionCluster",
			Func:      clusterSuite.ProvisionCluster,
			DependsOn: []string{"PopulateSampleData"},
		},
		{
			Name:      "CheckInstallation",
			Func:      installationSuite.CheckHealth,
			DependsOn: []string{"ProvisionCluster"},
		},
		{
			Name:      "DeleteInstallation",
			Func:      installationSuite.Cleanup,
			DependsOn: []string{"CheckInstallation"},
		},
		{
			Name:      "DeleteCluster",
			Func:      clusterSuite.DeleteCluster,
			DependsOn: []string{"DeleteInstallation"},
		},
	}
}
