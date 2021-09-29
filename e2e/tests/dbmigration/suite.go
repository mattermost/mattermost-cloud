// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//+build e2e

package dbmigration

import (
	"encoding/json"

	"github.com/mattermost/mattermost-cloud/e2e/pkg"
	"github.com/mattermost/mattermost-cloud/e2e/workflow"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/vrischmann/envconfig"
)

// TestConfig is test configuration coming from env vars.
type TestConfig struct {
	CloudURL                  string `envconfig:"default=http://localhost:8075"`
	DestinationDB             string
	InstallationDBType        string `envconfig:"default=aws-multitenant-rds-postgres"`
	InstallationFileStoreType string `envconfig:"default=bifrost"`
	Environment               string `envconfig:"default=dev"`
	Cleanup                   bool   `envconfig:"default=true"`
}

// Test holds all data required for a db migration test.
type Test struct {
	Logger   logrus.FieldLogger
	Suite    *workflow.DBMigrationSuite
	Workflow *workflow.Workflow
	Cleanup  bool
}

// SetupDBMigrationCommitTest sets up DB Migration tests which commits the migration.
func SetupDBMigrationCommitTest() (*Test, error) {
	logger := logrus.WithField("test", "db-migration-commit")

	config, err := readConfig(logger)
	if err != nil {
		return nil, err
	}

	suite, err := setupDBMigrationTestSuite(config, logger)
	if err != nil {
		return nil, err
	}
	work := commitDBMigrationWorkflow(suite)

	return &Test{
		Logger:   logger,
		Suite:    suite,
		Workflow: work,
		Cleanup:  config.Cleanup,
	}, nil
}

// SetupDBMigrationRollbackTest sets up DB Migration tests which rollbacks the migration.
func SetupDBMigrationRollbackTest() (*Test, error) {
	logger := logrus.WithField("test", "db-migration-rollback")

	config, err := readConfig(logger)
	if err != nil {
		return nil, err
	}

	suite, err := setupDBMigrationTestSuite(config, logger)
	if err != nil {
		return nil, err
	}
	work := rollbackDBMigrationWorkflow(suite)

	return &Test{
		Logger:   logger,
		Suite:    suite,
		Workflow: work,
		Cleanup:  config.Cleanup,
	}, nil
}

func readConfig(logger logrus.FieldLogger) (TestConfig, error) {
	var config TestConfig
	err := envconfig.Init(&config)
	if err != nil {
		return TestConfig{}, errors.Wrap(err, "unable to read environment configuration")
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return TestConfig{}, errors.Wrap(err, "failed to marshal config to json")
	}

	logger.Infof("Test Config: %s", configJSON)

	return config, nil
}

func setupDBMigrationTestSuite(config TestConfig, logger logrus.FieldLogger) (*workflow.DBMigrationSuite, error) {
	client := model.NewClient(config.CloudURL)

	params := workflow.DBMigrationSuiteParams{
		InstallationSuiteParams: workflow.InstallationSuiteParams{
			DBType:        config.InstallationDBType,
			FileStoreType: config.InstallationFileStoreType,
		},
		DestinationDBID: config.DestinationDB,
	}

	kubeClient, err := pkg.GetK8sClient()
	if err != nil {
		return nil, err
	}

	return workflow.NewDBMigrationSuite(params, config.Environment, client, kubeClient, logger), nil
}

// Run runs the test workflow.
func (w *Test) Run() error {
	err := workflow.RunWorkflow(w.Workflow, w.Logger)
	if err != nil {
		return errors.Wrap(err, "error running workflow")
	}
	return nil
}
