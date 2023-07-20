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
	return []*workflow.Step{
		{
			Name:              "CreateCluster",
			Func:              clusterSuite.CreateCluster,
			DependsOn:         []string{},
			GetExpectedEvents: clusterSuite.ClusterCreationEvents,
		},
		{
			Name:              "CreateInstallationWithVersionedAWSS3Filestore",
			Func:              installationSuite.CreateInstallationWithVersionedAWSS3Filestore,
			DependsOn:         []string{"CreateCluster"},
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
	}
}

func basicCreateDeleteInstallationSteps(clusterSuite *workflow.ClusterSuite, installationSuite *workflow.InstallationSuite) []*workflow.Step {
	return []*workflow.Step{
		{
			Name:              "CreateCluster",
			Func:              clusterSuite.CreateCluster,
			DependsOn:         []string{},
			GetExpectedEvents: clusterSuite.ClusterCreationEvents,
		},
		{
			Name:              "CreateInstallation",
			Func:              installationSuite.CreateInstallation,
			DependsOn:         []string{"CreateCluster"},
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
	}
}
