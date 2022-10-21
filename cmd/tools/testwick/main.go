// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	testwick "github.com/mattermost/mattermost-cloud/cmd/tools/testwick/wicker"
	cmodel "github.com/mattermost/mattermost-cloud/model"
	mmodel "github.com/mattermost/mattermost-server/v5/model"
	"github.com/moby/moby/pkg/namesgenerator"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/sirupsen/logrus"
)

type testerCfg struct {
	provisioner            string
	cloudImageName         string
	cloudImageTag          string
	cloudGroupID           string
	owner                  string
	installationSize       string
	installationAffinity   string
	installationAnnotation string
	installationDBType     string
	installationFilestore  string
	hostedZoneDomain       string
	samples                int
	channelSamples         int
	messagesSamples        int
	messagesSamplesSleep   time.Duration
	finalDeletion          bool
}

type populateDataCfg struct {
	provisioner          string
	owner                string
	channelSamples       int
	messagesSamples      int
	messagesSamplesSleep time.Duration
}

type deleteCfg struct {
	provisioner          string
	owner                string
	partial              bool
	messagesSamplesSleep time.Duration
	deletionSkipStep     int
}

var testerConfig = testerCfg{}
var populateDataConfig = populateDataCfg{}
var deleteConfig = deleteCfg{}

var logger *logrus.Logger

var rootCmd = &cobra.Command{
	Use:   "testwick",
	Short: "Used to test provisioner installations and interact with them.",
	// SilenceErrors allows us to explicitly log the error returned from rootCmd below.
	SilenceErrors: true,
}

// TODO add flag for password and user
// TODO add flag for group selection

func init() {
	logger = logrus.New()
	logger.Out = os.Stdout
	logger.Formatter = &logrus.JSONFormatter{}
	// Output to stdout instead of the default stderr.
	logger.SetOutput(os.Stdout)

	testerCmd.PersistentFlags().StringVar(&testerConfig.provisioner, "provisioner", "", "The url for the provisioner")
	testerCmd.PersistentFlags().StringVar(&testerConfig.cloudImageName, "provisioner-image-name", "mattermost/mm-ee-cloud", "The Cloud image will be used for provisioner")
	testerCmd.PersistentFlags().StringVar(&testerConfig.cloudImageTag, "provisioner-image-tag", "latest", "The Cloud image tag will be used for provisioner")
	testerCmd.PersistentFlags().StringVar(&testerConfig.cloudGroupID, "provisioner-group-id", "", "The provisioner Group to add the installations to")
	testerCmd.PersistentFlags().StringVar(&testerConfig.hostedZoneDomain, "hosted-zone", "", "The hosted zone you need to run your installations eg. test.mattermost.cloud")
	testerCmd.PersistentFlags().IntVar(&testerConfig.samples, "samples", 1, "The number of samples installations to interact with")
	testerCmd.PersistentFlags().IntVar(&testerConfig.channelSamples, "channel-samples", 1, "The number of channel samples to create in installation")
	testerCmd.PersistentFlags().IntVar(&testerConfig.messagesSamples, "channel-messages", 100, "The number of channel messages to post")
	testerCmd.PersistentFlags().DurationVar(&testerConfig.messagesSamplesSleep, "channel-messages-sleep", 10*time.Second, "The number of time to sleep before post the next message")
	testerCmd.PersistentFlags().StringVar(&testerConfig.owner, "owner", "testwick", "The owner of the installation. Prefer to keep the default")
	testerCmd.PersistentFlags().StringVar(&testerConfig.installationSize, "installation-size", "100users", "The size of the installation")
	testerCmd.PersistentFlags().StringVar(&testerConfig.installationAffinity, "installation-affinity", cmodel.InstallationAffinityMultiTenant, "The installation affinity type eg. multitenant")
	testerCmd.PersistentFlags().StringVar(&testerConfig.installationAnnotation, "installation-annotation", "uat", "The installation annotation to schedule to cluster type eg. uat")
	testerCmd.PersistentFlags().StringVar(&testerConfig.installationDBType, "installation-db-type", cmodel.InstallationDatabaseMultiTenantRDSPostgresPGBouncer, "The installation database type eg. aws-multitenant-rds-postgres")
	testerCmd.PersistentFlags().StringVar(&testerConfig.installationFilestore, "installation-filestore", cmodel.InstallationFilestoreBifrost, "The installation filestore type eg. bifrost")
	testerCmd.PersistentFlags().BoolVar(&testerConfig.finalDeletion, "final-deletion", true, "Whether to delete the installations after a successful testwick run")
	testerCmd.MarkFlagRequired("provisioner") //nolint
	testerCmd.MarkFlagRequired("hosted-zone") //nolint

	populateDataCmd.PersistentFlags().StringVar(&populateDataConfig.provisioner, "provisioner", "", "The url for the provisioner")
	populateDataCmd.PersistentFlags().StringVar(&populateDataConfig.owner, "owner", "testwick", "The owner of the testwick installations to populate data to")
	populateDataCmd.PersistentFlags().IntVar(&populateDataConfig.channelSamples, "channel-samples", 1, "The number of channel samples to create in installation")
	populateDataCmd.PersistentFlags().IntVar(&populateDataConfig.messagesSamples, "channel-messages", 100, "The number of channel messages to post")
	populateDataCmd.PersistentFlags().DurationVar(&populateDataConfig.messagesSamplesSleep, "channel-messages-sleep", 10*time.Second, "The number of time to sleep before post the next message")
	populateDataCmd.MarkFlagRequired("provisioner") //nolint
	populateDataCmd.MarkFlagRequired("owner")       //nolint

	deleteCmd.PersistentFlags().StringVar(&deleteConfig.provisioner, "provisioner", "", "The url for the provisioner")
	deleteCmd.PersistentFlags().StringVar(&deleteConfig.owner, "owner", "testwick", "The owner of the testwick installations to delete")
	deleteCmd.PersistentFlags().BoolVar(&deleteConfig.partial, "partial", false, "Whether to delete some or all installations")
	deleteCmd.PersistentFlags().DurationVar(&deleteConfig.messagesSamplesSleep, "installation-deletions-sleep", 10*time.Second, "The number of time to sleep before deleting the next installation")
	deleteCmd.PersistentFlags().IntVar(&deleteConfig.deletionSkipStep, "deletion-skip-step", 1, "The number of installations to skip in every deletion loop")
	deleteCmd.MarkFlagRequired("provisioner") //nolint
	deleteCmd.MarkFlagRequired("owner")       //nolint

	rootCmd.AddCommand(testerCmd)
	rootCmd.AddCommand(populateDataCmd)
	rootCmd.AddCommand(deleteCmd)

}

func main() {
	if err := rootCmd.Execute(); err != nil {
		logger.WithError(err).Error("command failed")
		os.Exit(1)
	}
}

var testerCmd = &cobra.Command{
	Use:   "test",
	Short: "Run tester to start creating installations with provisioner and interact with them.",
	Run: func(command *cobra.Command, args []string) {
		u, err := url.Parse(testerConfig.provisioner)
		if err != nil {
			logger.WithError(err).Error("Invalid provisioner URL")
			return
		}
		provisionerClient := cmodel.NewClient(u.String())
		g, _ := errgroup.WithContext(context.Background())

		for i := 0; i < testerConfig.samples; i++ {
			name := normalizeDNS(namesgenerator.GetRandomName(5))
			dns := fmt.Sprintf("%s.%s", name, testerConfig.hostedZoneDomain)
			logger.Infof("DNS is %s", dns)
			installation, err := provisionerClient.GetInstallationByDNS(dns, &cmodel.GetInstallationRequest{})
			if err != nil {
				logger.WithError(err).Errorf("Failed to check if installation with DNS %s already exists", dns)
				return
			}

			for installation != nil {
				logger.Infof("Installation with DNS %s already exists. Choosing another random name...", dns)
				name = normalizeDNS(namesgenerator.GetRandomName(5))
				dns = fmt.Sprintf("%s.%s", name, testerConfig.hostedZoneDomain)
				installation, err = provisionerClient.GetInstallationByDNS(dns, &cmodel.GetInstallationRequest{})
				if err != nil {
					logger.WithError(err).Errorf("Failed to check if installation with DNS %s already exists", dns)
					return
				}
			}

			g.Go(func() error {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer cancel()

				mmClient := mmodel.NewAPIv4Client(fmt.Sprintf("https://%s", dns))
				testwicker := testwick.NewTestWicker(provisionerClient, mmClient, logger)

				workflow := testwick.NewWorkflow(logger)
				workflow.AddStep(testwick.Step{
					Name: "CreateInstallation",
					Func: testwicker.CreateInstallation(&cmodel.CreateInstallationRequest{
						Name:        name,
						DNSNames:    []string{dns},
						OwnerID:     testerConfig.owner,
						Size:        testerConfig.installationSize,
						Affinity:    testerConfig.installationAffinity,
						Database:    testerConfig.installationDBType,
						Filestore:   testerConfig.installationFilestore,
						Image:       testerConfig.cloudImageName,
						Version:     testerConfig.cloudImageTag,
						Annotations: []string{testerConfig.installationAnnotation},
						GroupID:     testerConfig.cloudGroupID,
					}),
				}, testwick.Step{
					Name: "WaitForInstallationStable",
					Func: testwicker.WaitForInstallationStable(),
				}, testwick.Step{
					Name: "IsInstallationUp",
					Func: testwicker.IsUP(),
				}, testwick.Step{
					Name: "SetupInstallation",
					Func: testwicker.SetupInstallation(),
				}, testwick.Step{
					Name: "CreateTeam",
					Func: testwicker.CreateTeam(),
				}, testwick.Step{
					Name: "AddTeamMember",
					Func: testwicker.AddTeamMember(),
				})

				for j := 0; j < testerConfig.channelSamples; j++ {
					workflow.AddStep(testwick.Step{
						Name: "CreateChannel",
						Func: testwicker.CreateChannel(20),
					}, testwick.Step{
						Name: "CreateIncomingWebhook",
						Func: testwicker.CreateIncomingWebhook(),
					}, testwick.Step{
						Name: "PostMessages",
						Func: testwicker.PostMessage(testerConfig.messagesSamples, testerConfig.messagesSamplesSleep),
					})
				}

				if testerConfig.finalDeletion {
					workflow.AddStep(testwick.Step{
						Name: "DeleteInstallation",
						Func: testwicker.DeleteInstallation(),
					})
				}

				if err := workflow.Run(ctx, testwicker); err != nil {
					testwicker.DeleteInstallation()
					cancel()
					return err
				}

				return nil
			})
		}
		if err := g.Wait(); err != nil {
			logger.WithError(err).Error()
			return
		}
	},
}

var populateDataCmd = &cobra.Command{
	Use:   "populate-data",
	Short: "Run populate data to interact with existing installations.",
	Run: func(command *cobra.Command, args []string) {
		u, err := url.Parse(populateDataConfig.provisioner)
		if err != nil {
			logger.WithError(err).Error("Invalid provisioner URL")
			return
		}
		provisionerClient := cmodel.NewClient(u.String())
		g, _ := errgroup.WithContext(context.Background())
		logger.Infof("Getting all existing installations owned by %s", populateDataConfig.owner)
		installations, err := provisionerClient.GetInstallations(&cmodel.GetInstallationsRequest{
			OwnerID: populateDataConfig.owner,
			Paging: cmodel.Paging{
				PerPage: 10000,
			},
		})
		if err != nil {
			logger.WithError(err).Error("Failed to get owner installations")
			return
		}

		for _, installation := range installations {
			installation := installation
			logger.Infof("Populating data for installation %s", installation.DNS)
			g.Go(func() error {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer cancel()
				mmClient := mmodel.NewAPIv4Client(fmt.Sprintf("https://%s", installation.DNS))
				testwicker := testwick.NewTestWicker(provisionerClient, mmClient, logger)
				workflow := testwick.NewWorkflow(logger)

				err = testwicker.UpdateTestwickWithExistingInstallation(installation)
				if err != nil {
					return errors.Wrap(err, "Failed to update testwick with existing installation data")
				}

				for j := 0; j < populateDataConfig.channelSamples; j++ {
					workflow.AddStep(testwick.Step{
						Name: "CreateChannel",
						Func: testwicker.CreateChannel(20),
					}, testwick.Step{
						Name: "CreateIncomingWebhook",
						Func: testwicker.CreateIncomingWebhook(),
					}, testwick.Step{
						Name: "PostMessages",
						Func: testwicker.PostMessage(populateDataConfig.messagesSamples, populateDataConfig.messagesSamplesSleep),
					})
				}
				if err := workflow.Run(ctx, testwicker); err != nil {
					cancel()
					return err
				}

				return nil
			})
		}
		if err := g.Wait(); err != nil {
			logger.WithError(err).Error()
			return
		}
	},
}

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Run delete to clean all installations created with testwick.",
	Run: func(command *cobra.Command, args []string) {
		u, err := url.Parse(deleteConfig.provisioner)
		if err != nil {
			logger.WithError(err).Error("Invalid provisioner URL")
			return
		}
		provisionerClient := cmodel.NewClient(u.String())

		installations, err := provisionerClient.GetInstallations(&cmodel.GetInstallationsRequest{
			OwnerID: deleteConfig.owner,
			Paging: cmodel.Paging{
				PerPage: 10000,
			},
		})
		if err != nil {
			logger.WithError(err).Error("Failed to get owner installations")
			return
		}

		if deleteConfig.partial {
			var step int
			if deleteConfig.deletionSkipStep != 0 {
				step = deleteConfig.deletionSkipStep
			} else {
				step = 1
			}
			i := 1
			for i <= len(installations) {
				logger.Infof("Deleting installation %s", installations[i].ID)
				err = provisionerClient.DeleteInstallation(installations[i].ID)
				if err != nil {
					logger.WithError(err).Errorf("Failed to delete installation %s", installations[i].ID)
					return
				}
				i = i + step
				time.Sleep(deleteConfig.messagesSamplesSleep)
			}
		} else {
			for _, installation := range installations {
				logger.Infof("Deleting installation %s", installation.ID)
				err = provisionerClient.DeleteInstallation(installation.ID)
				if err != nil {
					logger.WithError(err).Errorf("Failed to delete installation %s", installation.ID)
					return
				}
			}
			time.Sleep(deleteConfig.messagesSamplesSleep)
		}
	},
}

func normalizeDNS(s string) string {
	re := regexp.MustCompile(`[^(a-zA-Z0-0\-)]+`)
	x := re.ReplaceAllString(s, "")
	return strings.Trim(x, "")
}
