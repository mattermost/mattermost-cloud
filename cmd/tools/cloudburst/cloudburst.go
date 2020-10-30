// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	cloud "github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"
	log "github.com/sirupsen/logrus"
)

var logger *log.Logger

type completedReport struct {
	installation *cloud.Installation
	completedAt  time.Time
	createdAt    time.Time
}

type errorReport struct {
	installation *cloud.Installation
	timestamp    time.Time
	message      string
}

type ReportType string

const (
	ErrorReportType     ReportType = "errorReport"
	CompletedReportType ReportType = "completedReport"
)

type Report interface {
	Type() ReportType
}

func (e *errorReport) Type() ReportType {
	return ErrorReportType
}

func (c *completedReport) Type() ReportType {
	return CompletedReportType
}

type Blaster struct {
	client  *cloud.Client
	testID  string
	group   *cloud.Group
	reports []Report
}

func NewBlaster(serverAddress string) *Blaster {
	client := cloud.NewClient(serverAddress)
	return &Blaster{client: client, testID: cloud.NewID()}
}

var blastCommand = &cobra.Command{
	Use:   "blast",
	Short: "Run a load test against a provisioning server.",
	RunE: func(cmd *cobra.Command, args []string) error {
		serverAddress, _ := cmd.Flags().GetString("server")
		batchSize, _ := cmd.Flags().GetInt("batch")
		total, _ := cmd.Flags().GetInt("total")
		runs, _ := cmd.Flags().GetInt("runs")

		logger.SetLevel(log.DebugLevel)
		logger.Infof("Server address %s", serverAddress)
		logger.WithField("server", serverAddress)
		blaster := NewBlaster(serverAddress)
		err := blaster.createGroup()
		if err != nil {
			return errors.Wrap(err, "failed to create group for test installations")
		}

		logger.Infof("Waiting for Group to be created..")
		blaster.waitForGroup()

		allReports := []Report{}
		for i := 0; i < runs; i++ {
			logger.Infof("Requesting Installations be created..")
			created := blaster.createInstallations(total, batchSize)
			logger.Infof("Waiting for Installations to reconcile..")
			blaster.waitForInstallations(created)
			logger.Infof("Cleaning up Installations...")
			blaster.cleanupInstallations(created)
			report := blaster.compileReports()
			logger.Infof(
				"\nCompleted test:\n\nErrors: %d\nSuccessful Installs: %d\nMinimum Time to Reconcile: %d seconds\nMedian Time to Reconcile: %d seconds\nMaximum Time to Reconcile: %d seconds\n\n",
				report.errorCount,
				report.successCount,
				report.minDuration,
				report.medianDuration,
				report.maxDuration,
			)
			allReports = append(allReports, blaster.reports...)
			blaster.reports = []Report{}
		}
		blaster.reports = allReports
		report := blaster.compileReports()
		logger.Infof(
			"\nCompleted test:\n\nErrors: %d\nSuccessful Installs: %d\nMinimum Time to Reconcile: %d seconds\nMedian Time to Reconcile: %d seconds\nMaximum Time to Reconcile: %d seconds\n\n",
			report.errorCount,
			report.successCount,
			report.minDuration,
			report.medianDuration,
			report.maxDuration,
		)

		return blaster.cleanupGroup()
	},
}

type results struct {
	errorCount     int64
	successCount   int64
	medianDuration int64
	maxDuration    int64
	minDuration    int64
}

type Durations []int64

func (d Durations) Len() int {
	return len(d)
}

func (d Durations) Less(i, j int) bool {
	return d[i] < d[j]
}

func (d Durations) Swap(i, j int) {
	d[i], d[j] = d[j], d[i]
}

func (b *Blaster) addReports(reports ...Report) {
	b.reports = append(b.reports, reports...)
}

func (b *Blaster) compileReports() (output *results) {
	output = new(results)
	durations := []int64{}
	for _, report := range b.reports {
		switch r := report.(type) {
		case *completedReport:
			durations = append(durations, r.completedAt.Unix()-r.createdAt.Unix())
			output.successCount++
		case *errorReport:
			output.errorCount++
		}
	}
	sorted := Durations(durations)
	sort.Sort(sorted)
	output.maxDuration = sorted[len(sorted)-1]
	output.minDuration = sorted[0]
	output.medianDuration = sorted[len(sorted)/2]
	return
}

func (b *Blaster) waitForInstallations(input map[string]*cloud.Installation) {
	waiting := make(map[string]*cloud.Installation)
	for _, i := range input {
		waiting[i.ID] = i
	}

	for {
		for _, w := range waiting {
			this, err := b.client.GetInstallation(w.ID, nil)
			if err != nil {
				logger.WithError(err).Warningf("failed to fetch Installation %s", w.ID)
				continue
			}
			if this == nil {
				b.addReports(&errorReport{
					installation: this.Installation,
					timestamp:    time.Now(),
					message:      err.Error(),
				})
				logger.Errorf("Installation %s has gone missing", w.ID)
				delete(waiting, this.ID)
				continue
			}
			if strings.Contains(this.State, "failed") {
				logger.Errorf("Installation %s failed to be created; not waiting for it", this.ID)
				b.addReports(&errorReport{
					installation: this.Installation,
					timestamp:    time.Now(),
					message:      err.Error(),
				})
				delete(waiting, this.ID)
				continue
			}
			if this.State != cloud.InstallationStateStable {
				continue
			}
			r := &completedReport{
				installation: this.Installation,
				createdAt:    time.Unix(this.CreateAt/1000, 0),
				completedAt:  time.Now(),
			}
			b.addReports(r)
			log.Infof(
				"Creation time for %s was: %d seconds",
				r.installation.ID,
				r.completedAt.Unix()-r.createdAt.Unix())
			delete(waiting, this.ID)
		}
		if len(waiting) == 0 {
			return
		} else {
			time.Sleep(5 * time.Second)
		}
	}
}

func (b *Blaster) waitForGroup() {
	for {
		g, err := b.client.GetGroup(b.group.ID)
		if err != nil {
			logger.WithError(err).Errorf("failed to get group %s", b.group.ID)
			continue
		}
		if g == nil {
			continue
		}
		return
	}
}
func init() {
	logger = log.New()
	blastCommand.PersistentFlags().String("server", "http://localhost:8075", "Location of the Provisioning Server to load test")
	blastCommand.PersistentFlags().Int("runs", 1, "Number of times to repeat the test")
	blastCommand.PersistentFlags().Int("batch", 5, "Specify the number of Installations in each batch. Installations in a batch are install serially and batches are installed in parallel.")
	blastCommand.PersistentFlags().Int("total", 20, "Number of Installations to provision")
}

func (b *Blaster) serialBatchInstall(count int) (installations []*cloud.Installation) {
	for i := 0; i < count; i++ {
		install, err := b.createInstallation()
		if err != nil {
			logger.WithError(err).Warn("failed to request Installation creation")
			i--
			continue
		}
		logger.Infof("Requested creation successfully: %s", install.ID)
		installations = append(installations, install)
	}
	return
}

func (b *Blaster) createGroup() error {
	group, err := b.client.CreateGroup(
		&cloud.CreateGroupRequest{
			Name:            b.testID,
			Description:     fmt.Sprintf("Load Test Group for Test %s", b.testID),
			APISecurityLock: false,
		})
	if err != nil {
		return errors.Wrapf(err, "failed to create a group for test %s", b.testID)
	}
	b.group = group
	return nil
}

func (b *Blaster) cleanupInstallations(installations map[string]*cloud.Installation) {
	for len(installations) > 0 {
		for _, install := range installations {
			fetched, err := b.client.GetInstallation(install.ID, nil)
			if err != nil {
				logger.WithError(err).Warnf("failed to look up Installation %s", install.ID)
				continue
			}
			if fetched == nil {
				logger.Warnf("Installation %s not found; will not retry deletion", install.ID)
				b.addReports(&errorReport{
					installation: install,
					timestamp:    time.Now(),
					message:      fmt.Sprintf("%s not found", install.ID),
				})
				delete(installations, install.ID)
				continue
			}
			switch fetched.State {
			case cloud.InstallationStateDeleted:
				logger.Infof("Successfully deleted Installation %s", fetched.ID)
				delete(installations, fetched.ID)
			case cloud.InstallationStateDeletionRequested:
				fallthrough
			case cloud.InstallationStateDeletionFinalCleanup:
				fallthrough
			case cloud.InstallationStateDeletionInProgress:
			default:
				b.client.DeleteInstallation(fetched.ID)
			}
		}
	}
}

func (b *Blaster) cleanupGroup() error {
	err := b.client.DeleteGroup(b.group.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to delete group %s", b.group.ID)
	}
	return nil
}

func (b *Blaster) createInstallations(total, batchSize int) map[string]*cloud.Installation {
	installationsChannel := make(chan []*cloud.Installation)
	for i := 0; i < total; i += batchSize {
		go func(out chan []*cloud.Installation, batchNumber int) {
			batch := b.serialBatchInstall(batchSize)
			out <- batch
		}(installationsChannel, i)
	}

	allInstallations := map[string]*cloud.Installation{}
	for i := 0; i < total/batchSize; i++ {
		batch := <-installationsChannel
		for _, installation := range batch {
			allInstallations[installation.ID] = installation
		}
	}

	return allInstallations
}

func (b *Blaster) createInstallation() (*cloud.Installation, error) {
	installationDTO, err := b.client.CreateInstallation(
		&cloud.CreateInstallationRequest{
			OwnerID:         b.testID,
			GroupID:         b.group.ID,
			Database:        cloud.InstallationDatabaseMultiTenantRDSPostgres,
			Filestore:       cloud.InstallationFilestoreMultiTenantAwsS3,
			Size:            mmv1alpha1.Size1000String,
			Affinity:        cloud.InstallationAffinityMultiTenant,
			DNS:             fmt.Sprintf("%s-%s.loadtest.dev.cloud.mattermost.com", b.testID, cloud.NewID()[:6]),
			APISecurityLock: false,
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Installation")
	}
	return installationDTO.Installation, nil
}

func main() {
	if err := blastCommand.Execute(); err != nil {
		logger.WithError(err).Error("command failed")
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:   "cloudburst",
	Short: "Cloudburst is a tool for testing explosive demand against the Cloud Provisioner",
	Run: func(cmd *cobra.Command, args []string) {
		blastCommand.RunE(cmd, args)
	},
	// SilenceErrors allows us to explicitly log the error returned from rootCmd below.
	SilenceErrors: true,
}
