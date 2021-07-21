// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//+build e2e

package workflow

import (
	"context"

	"github.com/mattermost/mattermost-cloud/e2e/pkg"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
)

// NewInstallationSuite creates new Installation testing suite.
func NewInstallationSuite(params InstallationSuiteParams, env string, client *model.Client, kubeClient kubernetes.Interface, logger logrus.FieldLogger) *InstallationSuite {
	return &InstallationSuite{
		client:     client,
		kubeClient: kubeClient,
		logger:     logger.WithField("suite", "installation"),
		env:        env,
		Params:     params,
		Meta:       InstallationSuiteMeta{},
	}
}

// InstallationSuite is testing suite for Installations.
type InstallationSuite struct {
	client     *model.Client
	kubeClient kubernetes.Interface
	logger     logrus.FieldLogger
	env        string

	Params InstallationSuiteParams
	Meta   InstallationSuiteMeta
}

// InstallationSuiteParams contains parameters for InstallationSuite.
type InstallationSuiteParams struct {
	DBType        string
	FileStoreType string
}

// InstallationSuiteMeta contains metadata for InstallationSuite.
type InstallationSuiteMeta struct {
	InstallationID        string
	InstallationDNS       string
	ClusterInstallationID string
	ConnectionString      string
	BulkExportStats       pkg.ExportStats
}

// CreateInstallation creates new Installation and waits for it to reach stable state.
func (w *InstallationSuite) CreateInstallation(ctx context.Context) error {
	if w.Meta.InstallationID == "" {
		installationBuilder := pkg.NewInstallationBuilderWithDefaults().
			DNS(pkg.GetDNS(w.env)).
			DB(w.Params.DBType).
			FileStore(w.Params.FileStoreType)

		installation, err := w.client.CreateInstallation(installationBuilder.CreateRequest())
		if err != nil {
			return errors.Wrap(err, "while creating installation")
		}
		w.logger.Infof("Installation created: %s", installation.ID)
		w.Meta.InstallationID = installation.ID
		w.Meta.InstallationDNS = installation.DNS
	}

	err := pkg.WaitForStable(w.client, w.Meta.InstallationID, w.logger)
	if err != nil {
		return errors.Wrap(err, "while waiting for installation creation")
	}

	err = pkg.WaitForInstallationAvailability(w.Meta.InstallationDNS, w.logger)
	if err != nil {
		return errors.Wrap(err, "while waiting for installation DNS")
	}

	return nil
}

// GetCI gets ClusterInstallation for an Installation being the part of test suite and saves it in metadata.
func (w *InstallationSuite) GetCI(ctx context.Context) error {
	ci, err := w.client.GetClusterInstallations(&model.GetClusterInstallationsRequest{InstallationID: w.Meta.InstallationID, Paging: model.AllPagesNotDeleted()})
	if err != nil {
		return errors.Wrap(err, "while getting CI")
	}
	w.Meta.ClusterInstallationID = ci[0].ID

	return nil
}

// GetConnectionStrAndExport saves db connection string currently configured for installation and export statistics.
func (w *InstallationSuite) GetConnectionStrAndExport(ctx context.Context) error {
	connectionString, err := pkg.GetConnectionString(w.client, w.Meta.ClusterInstallationID)
	if err != nil {
		return errors.Wrap(err, "while getting connection str")
	}
	w.Meta.ConnectionString = connectionString

	exportStats, err := pkg.GetBulkExportStats(w.client, w.kubeClient, w.Meta.ClusterInstallationID, w.Meta.InstallationID, w.logger)
	if err != nil {
		return errors.Wrap(err, "while getting CSV export")
	}
	w.Meta.BulkExportStats = exportStats
	w.logger.Infof("Bulk export stats: %v", exportStats)

	return nil
}

// PopulateSampleData populates installation with sample data.
func (w *InstallationSuite) PopulateSampleData(ctx context.Context) error {
	// Do not generate guest user as by default guest accounts are disabled,
	// which results in guest users being deactivated when Mattermost restarts.
	_, err := w.client.RunMattermostCLICommandOnClusterInstallation(w.Meta.ClusterInstallationID, []string{"sampledata", "--teams", "4", "--channels-per-team", "15", "--guests", "0"})
	if err != nil {
		return errors.Wrap(err, "while populating sample data for CI")
	}
	w.logger.Info("Sample data generated")

	return nil
}

// HibernateInstallation hibernates installation and waits for it to get hibernated.
func (w *InstallationSuite) HibernateInstallation(ctx context.Context) error {
	installation, err := w.client.GetInstallation(w.Meta.InstallationID, &model.GetInstallationRequest{})
	if err != nil {
		return errors.Wrap(err, "while getting installation to hibernate")
	}
	if installation.State == model.InstallationStateHibernating {
		w.logger.Info("installation already hibernating")
		return nil
	}

	installation, err = w.client.HibernateInstallation(w.Meta.InstallationID)
	if err != nil {
		return errors.Wrap(err, "while hibernating installation")
	}

	err = pkg.WaitForHibernation(w.client, w.Meta.InstallationID, w.logger)
	if err != nil {
		return errors.Wrap(err, "while waiting for installation to hibernate")
	}

	return nil
}

// WakeUpInstallation wakes up installation and waits for it to reach stable state.
func (w *InstallationSuite) WakeUpInstallation(ctx context.Context) error {
	installation, err := w.client.GetInstallation(w.Meta.InstallationID, &model.GetInstallationRequest{})
	if err != nil {
		return errors.Wrap(err, "while getting installation to wake up")
	}
	if installation.State == model.InstallationStateStable {
		w.logger.Info("installation already woken up")
		return nil
	}

	if installation.State == model.InstallationStateHibernating {
		installation, err = w.client.WakeupInstallation(w.Meta.InstallationID, nil)
		if err != nil {
			return errors.Wrap(err, "while waking up installation")
		}
	}

	if installation.State != model.InstallationStateWakeUpRequested &&
		installation.State != model.InstallationStateUpdateInProgress {
		return errors.Errorf("installation is in unexpected state: %s", installation.State)
	}

	err = pkg.WaitForStable(w.client, w.Meta.InstallationID, w.logger)
	if err != nil {
		return errors.Wrap(err, "while waiting for installation to wake up")
	}

	return nil
}

// Cleanup cleans up installation saved in suite metadata.
func (w *InstallationSuite) Cleanup(ctx context.Context) error {
	installation, err := w.client.GetInstallation(w.Meta.InstallationID, &model.GetInstallationRequest{})
	if err != nil {
		return errors.Wrap(err, "while getting installation to wake up")
	}
	if installation.State == model.InstallationStateDeleted {
		w.logger.Info("installation already deleted")
		return nil
	}
	if installation.State == model.InstallationStateDeletionRequested ||
		installation.State == model.InstallationStateDeletionInProgress ||
		installation.State == model.InstallationStateDeletionFinalCleanup {
		w.logger.Info("installation already marked for deletion")
		return nil
	}

	err = w.client.DeleteInstallation(w.Meta.InstallationID)
	if err != nil {
		return errors.Wrap(err, "while requesting installation removal")
	}

	return nil
}
