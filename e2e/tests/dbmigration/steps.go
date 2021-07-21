// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//+build e2e

package dbmigration

import (
	"github.com/mattermost/mattermost-cloud/e2e/workflow"
)

func baseMigrationSteps(dbMigSuite *workflow.DBMigrationSuite) []*workflow.Step {
	return []*workflow.Step{
		{
			Name:      "CreateInstallation",
			Func:      dbMigSuite.CreateInstallation,
			DependsOn: []string{},
		},
		{
			Name:      "GetMultiTenantDBID",
			Func:      dbMigSuite.GetMultiTenantDBID,
			DependsOn: []string{"CreateInstallation"},
		},
		{
			Name:      "GetCI",
			Func:      dbMigSuite.GetCI,
			DependsOn: []string{"CreateInstallation"},
		},
		{
			Name:      "PopulateSampleData",
			Func:      dbMigSuite.PopulateSampleData,
			DependsOn: []string{"GetCI"},
		},
		{
			Name:      "GetConnectionStrAndExport",
			Func:      dbMigSuite.GetConnectionStrAndExport,
			DependsOn: []string{"PopulateSampleData"},
		},
		{
			Name:      "HibernateInstallationBeforeMigration",
			Func:      dbMigSuite.HibernateInstallation,
			DependsOn: []string{"GetConnectionStrAndExport"},
		},
		{
			Name:      "RunDBMigration",
			Func:      dbMigSuite.RunDBMigration,
			DependsOn: []string{"HibernateInstallationBeforeMigration"},
		},
		{
			Name:      "WakeUpInstallationAfterMigration",
			Func:      dbMigSuite.WakeUpInstallation,
			DependsOn: []string{"RunDBMigration"},
		},
		{
			Name:      "AssertMigrationSuccessful",
			Func:      dbMigSuite.AssertMigrationSuccessful,
			DependsOn: []string{"WakeUpInstallationAfterMigration"},
		},
	}
}

func commitDBMigrationWorkflow(dbMigSuite *workflow.DBMigrationSuite) *workflow.Workflow {
	steps := baseMigrationSteps(dbMigSuite)

	steps = append(steps, &workflow.Step{
		Name:      "CommitMigration",
		Func:      dbMigSuite.CommitMigration,
		DependsOn: []string{"AssertMigrationSuccessful"},
	})

	return workflow.NewWorkflow(steps)
}

func rollbackDBMigrationWorkflow(dbMigSuite *workflow.DBMigrationSuite) *workflow.Workflow {
	steps := baseMigrationSteps(dbMigSuite)

	steps = append(steps, &workflow.Step{
		Name:      "HibernateInstallationBeforeRollback",
		Func:      dbMigSuite.HibernateInstallation,
		DependsOn: []string{"AssertMigrationSuccessful"},
	})
	steps = append(steps, &workflow.Step{
		Name:      "RollbackMigration",
		Func:      dbMigSuite.RollbackMigration,
		DependsOn: []string{"HibernateInstallationBeforeRollback"},
	})
	steps = append(steps, &workflow.Step{
		Name:      "WakeUpInstallationAfterRollback",
		Func:      dbMigSuite.WakeUpInstallation,
		DependsOn: []string{"RollbackMigration"},
	})
	steps = append(steps, &workflow.Step{
		Name:      "AssertRollbackSuccessful",
		Func:      dbMigSuite.AssertRollbackSuccessful,
		DependsOn: []string{"WakeUpInstallationAfterRollback"},
	})

	return workflow.NewWorkflow(steps)
}
