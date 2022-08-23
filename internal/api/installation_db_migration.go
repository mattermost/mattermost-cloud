// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api

import (
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/mattermost/mattermost-cloud/internal/common"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/webhook"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
)

// initInstallationDBMigration registers installation migration operation endpoints on the given router.
func initInstallationDBMigration(apiRouter *mux.Router, context *Context) {
	addContext := func(handler contextHandlerFunc, name string) *contextHandler {
		return newContextHandler(context, handler, name)
	}

	migrationsRouter := apiRouter.PathPrefix("/operations/database/migrations").Subrouter()

	migrationsRouter.Handle("", addContext(handleTriggerInstallationDatabaseMigration, "handleTriggerInstallationDatabaseMigration")).Methods("POST")
	migrationsRouter.Handle("", addContext(handleGetInstallationDBMigrationOperations, "handleGetInstallationDBMigrationOperations")).Methods("GET")

	migrationRouter := apiRouter.PathPrefix("/operations/database/migration/{migration:[A-Za-z0-9]{26}}").Subrouter()
	migrationRouter.Handle("", addContext(handleGetInstallationDBMigrationOperation, "handleGetInstallationDBMigrationOperation")).Methods("GET")
	migrationRouter.Handle("/commit", addContext(handleCommitInstallationDatabaseMigration, "handleCommitInstallationDatabaseMigration")).Methods("POST")
	migrationRouter.Handle("/rollback", addContext(handleRollbackInstallationDatabaseMigration, "handleRollbackInstallationDatabaseMigration")).Methods("POST")
}

// handleTriggerInstallationDatabaseMigration responds to POST /api/installations/operations/database/migrations,
// requests migration of Installation's data to different DB cluster.
func handleTriggerInstallationDatabaseMigration(c *Context, w http.ResponseWriter, r *http.Request) {
	c.Logger = c.Logger.WithField("action", "migrate-installation-database")

	migrationRequest, err := model.NewInstallationDBMigrationRequestFromReader(r.Body)
	if err != nil {
		c.Logger.WithError(err).Error("failed to decode request")
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	c.Logger = c.Logger.WithField("installation", migrationRequest.InstallationID)

	newState := model.InstallationStateDBMigrationInProgress

	installationDTO, status, unlockOnce := getInstallationForTransition(c, migrationRequest.InstallationID, newState)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	dbMigrations, err := c.Store.GetInstallationDBMigrationOperations(&model.InstallationDBMigrationFilter{
		Paging:         model.AllPagesNotDeleted(),
		InstallationID: installationDTO.ID,
		States:         []model.InstallationDBMigrationOperationState{model.InstallationDBMigrationStateSucceeded},
	})
	if err != nil {
		c.Logger.WithError(err).Error("Failed to query succeeded installation DB migrations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if len(dbMigrations) > 0 {
		c.Logger.Error("DB migration cannot be started if other successful migration is not committed")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	currentDB, err := c.Store.GetMultitenantDatabaseForInstallationID(installationDTO.ID)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to get current multi-tenant database for installation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	err = validateDBMigration(c, installationDTO.Installation, migrationRequest, currentDB)
	if err != nil {
		c.Logger.WithError(err).Errorf("Cannot migrate installation database")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	dbMigrationOperation := &model.InstallationDBMigrationOperation{
		InstallationID:         migrationRequest.InstallationID,
		SourceDatabase:         installationDTO.Database,
		DestinationDatabase:    migrationRequest.DestinationDatabase,
		SourceMultiTenant:      &model.MultiTenantDBMigrationData{DatabaseID: currentDB.ID},
		DestinationMultiTenant: migrationRequest.DestinationMultiTenant,
	}

	oldInstallationState := installationDTO.State

	dbMigrationOperation, err = c.Store.TriggerInstallationDBMigration(dbMigrationOperation, installationDTO.Installation)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to trigger DB migration operation")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	webhookPayload := &model.WebhookPayload{
		Type:      model.TypeInstallationDBMigration,
		ID:        dbMigrationOperation.ID,
		NewState:  string(dbMigrationOperation.State),
		OldState:  "n/a",
		Timestamp: time.Now().UnixNano(),
		ExtraData: map[string]string{"Installation": dbMigrationOperation.InstallationID, "Environment": c.Environment},
	}
	err = webhook.SendToAllWebhooks(c.Store, webhookPayload, c.Logger.WithField("webhookEvent", webhookPayload.NewState))
	if err != nil {
		c.Logger.WithError(err).Error("Unable to process and send webhooks")
	}

	err = c.EventProducer.ProduceInstallationStateChangeEvent(installationDTO.Installation, oldInstallationState)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to create installation state change event")
	}

	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, dbMigrationOperation)
}

// handleGetInstallationDBMigrationOperations responds to GET /api/installations/operations/database/migrations,
// returns list of installation migration operation.
func handleGetInstallationDBMigrationOperations(c *Context, w http.ResponseWriter, r *http.Request) {
	c.Logger = c.Logger.
		WithField("action", "list-installation-db-migrations")

	paging, err := parsePaging(r.URL)
	if err != nil {
		c.Logger.WithError(err).Error("failed to parse paging parameters")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	installationID := r.URL.Query().Get("installation")
	state := r.URL.Query().Get("state")
	var states []model.InstallationDBMigrationOperationState
	if state != "" {
		states = append(states, model.InstallationDBMigrationOperationState(state))
	}

	dbMigrations, err := c.Store.GetInstallationDBMigrationOperations(&model.InstallationDBMigrationFilter{
		Paging:         paging,
		InstallationID: installationID,
		States:         states,
	})
	if err != nil {
		c.Logger.WithError(err).Error("Failed to list installation migrations")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, dbMigrations)
}

// handleGetInstallationDBMigrationOperation responds to GET /api/installations/operations/database/migration/{migration},
// returns specified installation db migration operation.
func handleGetInstallationDBMigrationOperation(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	migrationID := vars["migration"]

	c.Logger = c.Logger.
		WithField("action", "get-installation-db-migration").
		WithField("migration-operation", migrationID)

	dbRestorationOp, err := c.Store.GetInstallationDBMigrationOperation(migrationID)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to get installation db migration")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if dbRestorationOp == nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, dbRestorationOp)
}

// handleCommitInstallationDatabaseMigration responds to POST /api/installations/operations/database/migration/{migration}/commit,
// commits database migration.
func handleCommitInstallationDatabaseMigration(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	migrationID := vars["migration"]

	c.Logger = c.Logger.WithField("action", "commit-installation-database-migration").
		WithField("migration-operation", migrationID)

	dbMigrationOperation, status, unlockOnce := lockInstallationDBMigrationOperation(c, migrationID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	if dbMigrationOperation.State != model.InstallationDBMigrationStateSucceeded {
		c.Logger.Warn("Cannot commit DB migration that hasn't succeeded")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	sourceDB := c.DBProvider.GetDatabase(dbMigrationOperation.InstallationID, dbMigrationOperation.SourceDatabase)

	err := sourceDB.TeardownMigrated(c.Store, dbMigrationOperation, c.Logger)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to tear down migrated database")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	dbMigrationOperation.State = model.InstallationDBMigrationStateCommitted
	err = c.Store.UpdateInstallationDBMigrationOperationState(dbMigrationOperation)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to set operation status to committed")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	outputJSON(c, w, dbMigrationOperation)
}

// handleRollbackInstallationDatabaseMigration responds to POST /api/installations/operations/database/migration/{migration}/rollback,
// rollbacks database migration.
func handleRollbackInstallationDatabaseMigration(c *Context, w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	migrationID := vars["migration"]

	c.Logger = c.Logger.WithField("action", "rollback-installation-database-migration").
		WithField("migration-operation", migrationID)

	dbMigrationOperation, status, unlockOnce := lockInstallationDBMigrationOperation(c, migrationID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockOnce()

	newState := model.InstallationDBMigrationStateRollbackRequested

	if !dbMigrationOperation.ValidTransitionState(newState) {
		c.Logger.Warn("Cannot rollback migration, invalid state")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	installationDTO, status, unlockInstOnce := lockInstallation(c, dbMigrationOperation.InstallationID)
	if status != 0 {
		w.WriteHeader(status)
		return
	}
	defer unlockInstOnce()

	if installationDTO.State != model.InstallationStateHibernating {
		c.Logger.Error("Installation needs to be hibernated to be rolled back")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	err := c.Store.TriggerInstallationDBMigrationRollback(dbMigrationOperation, installationDTO.Installation)
	if err != nil {
		c.Logger.WithError(err).Error("Failed to trigger db migration rollback")
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	unlockOnce()
	c.Supervisor.Do()

	w.WriteHeader(http.StatusAccepted)
	outputJSON(c, w, dbMigrationOperation)
}

func validateDBMigration(c *Context, installation *model.Installation, migrationRequest *model.InstallationDBMigrationRequest, currentDB *model.MultitenantDatabase) error {
	if migrationRequest.DestinationDatabase != model.InstallationDatabaseMultiTenantRDSPostgres ||
		installation.Database != model.InstallationDatabaseMultiTenantRDSPostgres {
		return errors.Errorf("db migration is supported only when both source and destination are %q database types", model.InstallationDatabaseMultiTenantRDSPostgres)
	}

	if migrationRequest.DestinationMultiTenant == nil {
		return errors.New("destination database data not provided")
	}

	destinationDB, err := c.Store.GetMultitenantDatabase(migrationRequest.DestinationMultiTenant.DatabaseID)
	if err != nil {
		return errors.Wrap(err, "failed to get destination multi-tenant database")
	}
	if destinationDB == nil {
		return errors.Errorf("destination database with id %q not found", migrationRequest.DestinationMultiTenant.DatabaseID)
	}

	if currentDB.ID == destinationDB.ID {
		return errors.New("destination database is the same as current")
	}

	if currentDB.VpcID != destinationDB.VpcID {
		return errors.New("databases VPCs do not match, only migration inside the same VPC is supported")
	}

	err = common.ValidateDBMigrationDestination(c.Store, destinationDB, installation.ID, aws.DefaultRDSMultitenantDatabasePostgresCountLimit)
	if err != nil {
		return errors.Wrap(err, "destination database validation failed")
	}

	return nil
}
