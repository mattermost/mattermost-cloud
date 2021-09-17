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
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"

	"github.com/sirupsen/logrus"
)

type testerCfg struct {
	provisioner           string
	owner                 string
	installationSize      string
	installationAffinity  string
	installationDBType    string
	installationFilestore string
	hostedZoneDomain      string
	samples               int
	channelSamples        int
	messagesSamples       int
}

var testerConfig = testerCfg{}
var logger *logrus.Logger

var rootCmd = &cobra.Command{
	Use:   "testwick",
	Short: "Used to test provisioner installations and interact with them.",
	// SilenceErrors allows us to explicitly log the error returned from rootCmd below.
	SilenceErrors: true,
}

func init() {
	logger = logrus.New()
	logger.Out = os.Stdout
	logger.Formatter = &logrus.JSONFormatter{}
	// Output to stdout instead of the default stderr.
	logger.SetOutput(os.Stdout)

	testerCmd.PersistentFlags().StringVar(&testerConfig.provisioner, "provisioner", "", "The url for the provisioner")
	testerCmd.PersistentFlags().StringVar(&testerConfig.hostedZoneDomain, "hosted-zone", "", "The hosted zone you need to run your installations eg. test.mattermost.cloud")
	testerCmd.PersistentFlags().IntVar(&testerConfig.samples, "samples", 1, "The number of samples installations to interact with")
	testerCmd.PersistentFlags().IntVar(&testerConfig.samples, "channel-samples", 1, "The number of channel samples to create in installation")
	testerCmd.PersistentFlags().IntVar(&testerConfig.channelSamples, "channel-messages", 100, "The number of channel messages to post")
	testerCmd.PersistentFlags().StringVar(&testerConfig.owner, "owner", "testwick", "The owner of the installation. Prefer to keep the default")
	testerCmd.PersistentFlags().StringVar(&testerConfig.installationSize, "installation-size", "100users", "The size of the installation")
	testerCmd.PersistentFlags().StringVar(&testerConfig.installationAffinity, "installation-affinity", cmodel.InstallationAffinityMultiTenant, "The installation affinity type eg. multitenant")
	testerCmd.PersistentFlags().StringVar(&testerConfig.installationDBType, "installation-db-type", cmodel.InstallationDatabaseMultiTenantRDSPostgresPGBouncer, "The installation database type eg. aws-multitenant-rds-postgres")
	testerCmd.PersistentFlags().StringVar(&testerConfig.installationFilestore, "installation-filestore", cmodel.InstallationFilestoreBifrost, "The installation filestore type eg. bifrost")
	testerCmd.MarkFlagRequired("provisioner") //nolint
	testerCmd.MarkFlagRequired("hosted-zone") //nolint

	rootCmd.AddCommand(testerCmd)
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

			dnsName := fmt.Sprintf("%s.%s", normalizeDNS(namesgenerator.GetRandomName(5)), testerConfig.hostedZoneDomain)

			g.Go(func() error {
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
				defer cancel()

				mmClient := mmodel.NewAPIv4Client(fmt.Sprintf("https://%s", dnsName))
				testwicker := testwick.NewTestWicker(provisionerClient, mmClient, logger)

				workflow := testwick.NewWorkflow(logger)
				workflow.AddStep(testwick.Step{
					Name: "CreateInstallation",
					Func: testwicker.CreateInstallation(&cmodel.CreateInstallationRequest{
						DNS:       dnsName,
						OwnerID:   testerConfig.owner,
						Size:      testerConfig.installationSize,
						Affinity:  testerConfig.installationAffinity,
						Database:  testerConfig.installationDBType,
						Filestore: testerConfig.installationFilestore,
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
						Func: testwicker.PostMessage(testerConfig.messagesSamples),
					})
				}
				workflow.AddStep(testwick.Step{
					Name: "DeleteInstallation",
					Func: testwicker.DeleteInstallation(),
				})
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

func normalizeDNS(s string) string {
	re := regexp.MustCompile(`[^(a-zA-Z0-0\-)]+`)
	x := re.ReplaceAllString(s, "")
	return strings.Trim(x, "")
}
