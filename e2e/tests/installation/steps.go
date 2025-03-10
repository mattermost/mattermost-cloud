// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//go:build e2e
// +build e2e

package installation

import (
	"github.com/mattermost/mattermost-cloud/e2e/workflow"
)

func versionedS3BucketInstallationLifecycleSteps(clusterSuite *workflow.ClusterSuite, installationSuite *workflow.InstallationSuite) []*workflow.Step {
	var initialDepends []string
	var steps []*workflow.Step

	if !clusterSuite.Params.UseExistingCluster {
		initialDepends = []string{"CreateCluster"}
		steps = []*workflow.Step{
			{
				Name:              "CreateCluster",
				Func:              clusterSuite.CreateCluster,
				DependsOn:         []string{},
				GetExpectedEvents: clusterSuite.ClusterCreationEvents,
			},
		}
	}

	steps = append(steps, []*workflow.Step{
		{
			Name:              "CreateInstallationWithVersionedAWSS3Filestore",
			Func:              installationSuite.CreateInstallationWithVersionedAWSS3Filestore,
			DependsOn:         initialDepends,
			GetExpectedEvents: installationSuite.InstallationCreationEvents,
		},
		{
			Name:      "GetVersionedAWSS3FilestoreCI",
			Func:      installationSuite.GetCI,
			DependsOn: []string{"CreateInstallationWithVersionedAWSS3Filestore"},
		},
		{
			Name:      "CheckVersionedAWSS3FilestoreClusterInstallationStatus",
			Func:      installationSuite.CheckClusterInstallationStatus,
			DependsOn: []string{"GetVersionedAWSS3FilestoreCI"},
		},
		{
			Name:      "CheckVersionedAWSS3FilestoreInstallation",
			Func:      installationSuite.CheckHealth,
			DependsOn: []string{"CheckVersionedAWSS3FilestoreClusterInstallationStatus"},
		},
		{
			Name:              "DeleteVersionedAWSS3FilestoreInstallation",
			Func:              installationSuite.Cleanup,
			DependsOn:         []string{"CheckVersionedAWSS3FilestoreInstallation"},
			GetExpectedEvents: installationSuite.InstallationDeletionEvents,
		},
	}...)

	return steps
}

func largeInstallationSizeLifecycleSteps(clusterSuite *workflow.ClusterSuite, installationSuite *workflow.InstallationSuite) []*workflow.Step {
	var initialDepends []string
	var steps []*workflow.Step

	if !clusterSuite.Params.UseExistingCluster {
		initialDepends = []string{"CreateCluster"}
		steps = []*workflow.Step{
			{
				Name:              "CreateCluster",
				Func:              clusterSuite.CreateCluster,
				DependsOn:         []string{},
				GetExpectedEvents: clusterSuite.ClusterCreationEvents,
			},
		}
	}

	steps = append(steps, []*workflow.Step{
		{
			Name:              "CreateInstallationWithLargeSize",
			Func:              installationSuite.CreateInstallationWithCustomProvisionerSize,
			DependsOn:         initialDepends,
			GetExpectedEvents: installationSuite.InstallationCreationEvents,
		},
		{
			Name:      "GetLargeSizeInstallationCI",
			Func:      installationSuite.GetCI,
			DependsOn: []string{"CreateInstallationWithLargeSize"},
		},
		{
			Name:      "CheckLargeSizeInstallationStatus",
			Func:      installationSuite.CheckClusterInstallationStatus,
			DependsOn: []string{"GetLargeSizeInstallationCI"},
		},
		{
			Name:      "CheckLargeSizeInstallation",
			Func:      installationSuite.CheckHealth,
			DependsOn: []string{"CheckLargeSizeInstallationStatus"},
		},
		{
			Name:              "DeleteLargeSizeInstallation",
			Func:              installationSuite.Cleanup,
			DependsOn:         []string{"CheckLargeSizeInstallation"},
			GetExpectedEvents: installationSuite.InstallationDeletionEvents,
		},
	}...)

	return steps
}

func basicCreateDeleteInstallationSteps(clusterSuite *workflow.ClusterSuite, installationSuite *workflow.InstallationSuite) []*workflow.Step {
	var initialDepends []string
	var steps []*workflow.Step

	if !clusterSuite.Params.UseExistingCluster {
		initialDepends = []string{"CreateCluster"}
		steps = []*workflow.Step{
			{
				Name:              "CreateCluster",
				Func:              clusterSuite.CreateCluster,
				DependsOn:         []string{},
				GetExpectedEvents: clusterSuite.ClusterCreationEvents,
			},
		}
	}

	steps = append(steps, []*workflow.Step{
		{
			Name:              "CreateInstallation",
			Func:              installationSuite.CreateInstallation,
			DependsOn:         initialDepends,
			GetExpectedEvents: installationSuite.InstallationCreationEvents,
		},
		{
			Name:      "GetCI",
			Func:      installationSuite.GetCI,
			DependsOn: []string{"CreateInstallation"},
		},
		{
			Name:      "CheckClusterInstallationStatus",
			Func:      installationSuite.CheckClusterInstallationStatus,
			DependsOn: []string{"GetCI"},
		},
		{
			Name:      "PopulateSampleData",
			Func:      installationSuite.PopulateSampleData,
			DependsOn: []string{"CheckClusterInstallationStatus"},
		},
		{
			Name:      "CheckInstallation",
			Func:      installationSuite.CheckHealth,
			DependsOn: []string{"PopulateSampleData"},
		},
		{
			Name:              "DeleteInstallation",
			Func:              installationSuite.Cleanup,
			DependsOn:         []string{"CheckInstallation"},
			GetExpectedEvents: installationSuite.InstallationDeletionEvents,
		},
	}...)

	return steps
}
