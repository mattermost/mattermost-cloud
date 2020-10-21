// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	cloud "github.com/mattermost/mattermost-cloud/model"
	mmv1alpha1 "github.com/mattermost/mattermost-operator/apis/mattermost/v1alpha1"
	log "github.com/sirupsen/logrus"
)

var logger *log.Logger

type Blaster struct {
	client *cloud.Client
	testID string
	group  *cloud.Group
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
		logger.Infof("Requesting Installations be created..")
		created := blaster.createInstallations(4, 4)
		logger.Infof("Waiting for Installations to reconcile..")
		blaster.waitForInstallations(created)
		logger.Infof("Cleaning up Installations...")
		blaster.cleanupInstallations(created)
		logger.Infof("Removing the group...")
		return blaster.cleanupGroup()
	},
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

			}
			if this == nil {
				logger.Errorf("Installation %s has gone missing")
				delete(waiting, this.ID)
				continue
			}
			if strings.Contains(this.State, "failed") {
				logger.Errorf("Installation %s failed to be created; not waiting for it")
				delete(waiting, this.ID)
				continue
			}
			if this.State != cloud.InstallationStateStable {
				continue
			}
			delete(waiting, this.ID)
		}
		if len(waiting) == 0 {
			return
		} else {
			logger.Debugf("Waiting for %d installations to stabilize", len(waiting))
			time.Sleep(5 * time.Second)
		}
	}
}

func (b *Blaster) waitForGroup() {
	for {
		g, err := b.client.GetGroup(b.group.ID)
		if err != nil {
			logger.WithError(err).Error("failed to get group %s", b.group.ID)
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
}

func (b *Blaster) serialBatchInstall(count int) (installations []*cloud.Installation) {
	for i := 0; i < count; i++ {
		install, err := b.createInstallation()
		if err != nil {
			logger.WithError(err).Warn("failed to create Installation")
			i--
		}
		logger.Debugf("Requested creation successfully: %s", install.ID)
		installations = append(installations, install)
	}
	logger.Debugf("Successfully created %d installations", count)
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
			}
			if fetched == nil {
				logger.Warnf("Installation %s not found; will not retry deletion", install.ID)
				delete(installations, install.ID)
				continue
			}
			switch fetched.State {
			case cloud.InstallationStateDeleted:
				logger.Debugf("Successfully deleted Installation %s", fetched.ID)
				delete(installations, fetched.ID)
				continue
			case cloud.InstallationStateDeletionRequested:
				fallthrough
			case cloud.InstallationStateDeletionInProgress:
				continue
			default:
				err = b.client.DeleteInstallation(fetched.ID)
				if err != nil {
					logger.WithError(err).Warnf("failed to delete Installation %s", install.ID)
					continue
				}
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
			logger.Debugf("Installing batch %d", batchNumber)
			batch := b.serialBatchInstall(batchSize)
			out <- batch
		}(installationsChannel, i)
	}

	allInstallations := map[string]*cloud.Installation{}
	for i := 0; i < total/batchSize; i++ {
		logger.Debugf("Have %d need %d Installations. Receiving batch..", i, total)
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
