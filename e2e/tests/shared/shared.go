// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//go:build e2e
// +build e2e

package shared

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"

	"github.com/mattermost/mattermost-cloud/clusterdictionary"
	"github.com/mattermost/mattermost-cloud/e2e/pkg"
	"github.com/mattermost/mattermost-cloud/e2e/pkg/eventstest"
	"github.com/mattermost/mattermost-cloud/e2e/tests/state"
	"github.com/mattermost/mattermost-cloud/e2e/workflow"
	awsTools "github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/vrischmann/envconfig"
)

const (
	webhookSuccessfulMessage      = "Provisioner E2E tests passed"
	webhookFailedMessage          = "Provisioner E2E tests failed"
	webhookSuccessEmoji           = "large_green_circle"
	webhookFailedEmoji            = "red_circle"
	webhookAttachmentColorSuccess = "#009E60"
	webhookAttachmentColorError   = "#FF0000"
)

// TODO: we can further parametrize the test according to our needs

// TestConfig is test configuration coming from env vars.
type TestConfig struct {
	Provisioner               string `envconfig:"default=kops"`
	CloudURL                  string `envconfig:"default=http://localhost:8075"`
	UseExistingCluster        bool   `envconfig:"optional,default=false"`
	InstallationDBType        string `envconfig:"default=aws-rds-postgres"`
	InstallationFilestoreType string `envconfig:"default=bifrost"`
	DNSSubdomain              string `envconfig:"default=dev.cloud.mattermost.com"`
	WebhookAddress            string `envconfig:"default=http://localhost:11111"`
	EventListenerAddress      string `envconfig:"default=http://localhost:11112"`
	ArgoManagedUtilities      bool   `envconfig:"optional,default=false"`
	FetchAMI                  bool   `envconfig:"default=true"`
	KopsAMI                   string `envconfig:"optional"`
	VPC                       string `envconfig:"optional"`
	Cleanup                   bool   `envconfig:"default=true"`
	ClusterRoleARN            string `envconfig:"optional"`
	NodeRoleARN               string `envconfig:"optional"`
	ClusterID                 string `envconfig:"optional"`
	InstallationID            string `envconfig:"optional"`
	Debug                     bool   `envconfig:"optional,default=false"`
}

// Test holds all data required for a db migration test.
type Test struct {
	Logger            logrus.FieldLogger
	ProvisionerClient *model.Client
	Workflow          *workflow.Workflow
	Steps             []*workflow.Step
	ClusterSuite      *workflow.ClusterSuite
	InstallationSuite *workflow.InstallationSuite
	EventsRecorder    *eventstest.EventsRecorder
	WebhookCleanup    func() error
	Cleanup           bool
}

func SetupTestWithDefaults(testName string) (*Test, error) {
	testID := model.NewID()
	state.TestID = testID
	state.TestName = testName
	logger := logrus.WithFields(map[string]interface{}{
		"test":   testName,
		"testID": testID,
	})

	config, err := readConfig(logger)
	if err != nil {
		return nil, err
	}

	if config.Debug {
		logger.Logger.SetLevel(logrus.DebugLevel)
	}

	client := model.NewClient(config.CloudURL)
	testAnnotations := testAnnotations(testID)

	clusterParams, err := buildClusterSuiteParams(config, client, testAnnotations, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to build cluster test params")
	}

	installationParams := workflow.InstallationSuiteParams{
		DBType:        config.InstallationDBType,
		FileStoreType: config.InstallationFilestoreType,
	}
	if !config.UseExistingCluster {
		installationParams.Annotations = testAnnotations
	}

	kubeClient, err := pkg.GetK8sClient()
	if err != nil {
		return nil, err
	}

	subOwner := "e2e-test"

	// We need to be cautious with introducing some parallelism for tests especially on step level
	// as webhook event will be delivered to only one channel.
	webhookChan, cleanup, err := pkg.SetupTestWebhook(client, config.WebhookAddress, subOwner, logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to setup webhook")
	}

	clusterMeta := workflow.ClusterSuiteMeta{ClusterID: config.ClusterID}
	clusterSuite := workflow.NewClusterSuite(clusterParams, clusterMeta, client, webhookChan, logger)

	installationMeta := workflow.InstallationSuiteMeta{InstallationID: config.InstallationID}

	installationSuite := workflow.NewInstallationSuite(installationParams, installationMeta, config.DNSSubdomain, client, kubeClient, webhookChan, logger)

	eventsRecorder := eventstest.NewEventsRecorder(subOwner, config.EventListenerAddress, logger.WithField("component", "event-recorder"), eventstest.RecordAll)

	return &Test{
		Logger:            logger,
		ProvisionerClient: client,
		WebhookCleanup:    cleanup,
		ClusterSuite:      clusterSuite,
		InstallationSuite: installationSuite,
		EventsRecorder:    eventsRecorder,
		Cleanup:           config.Cleanup,
	}, nil
}

func testAnnotations(testID string) []string {
	return []string{"e2e-test-cluster-lifecycle", fmt.Sprintf("test-id-%s", testID)}
}

func buildClusterSuiteParams(config TestConfig, client *model.Client, testAnnotations []string, logger logrus.FieldLogger) (workflow.ClusterSuiteParams, error) {
	if config.UseExistingCluster {
		logger.Info("Tests configured to run on existing clusters")
		return workflow.ClusterSuiteParams{
			UseExistingCluster: true,
		}, nil
	}

	logger.Info("Tests configured to create new cluster")

	createClusterReq := &model.CreateClusterRequest{
		AllowInstallations: true,
		Annotations:        testAnnotations,
		AMI:                config.KopsAMI,
		VPC:                config.VPC,
		Provisioner:        config.Provisioner,
	}

	if config.ArgoManagedUtilities {
		createClusterReq.ArgocdClusterRegister = map[string]string{
			"cluster-type": "customer",
		}

		createClusterReq.DesiredUtilityVersions = map[string]*model.HelmUtilityVersion{
			model.NginxCanonicalName: {
				Chart: model.UnmanagedUtilityVersion,
			},
			model.NginxInternalCanonicalName: {
				Chart: model.UnmanagedUtilityVersion,
			},
			model.ThanosCanonicalName: {
				Chart: model.UnmanagedUtilityVersion,
			},
			model.NodeProblemDetectorCanonicalName: {
				Chart: model.UnmanagedUtilityVersion,
			},
			model.CloudproberCanonicalName: {
				Chart: model.UnmanagedUtilityVersion,
			},
			model.FluentbitCanonicalName: {
				Chart: model.UnmanagedUtilityVersion,
			},
			model.TeleportCanonicalName: {
				Chart: model.UnmanagedUtilityVersion,
			},
			model.PgbouncerCanonicalName: {
				Chart: model.UnmanagedUtilityVersion,
			},
			model.MetricsServerCanonicalName: {
				Chart: model.UnmanagedUtilityVersion,
			},
			model.VeleroCanonicalName: {
				Chart: model.UnmanagedUtilityVersion,
			},
		}
	}

	if config.Provisioner == model.ProvisionerEKS {
		createClusterReq.ClusterRoleARN = config.ClusterRoleARN
		createClusterReq.NodeRoleARN = config.NodeRoleARN
	}

	// If specified, we fetch AMI from existing clusters.
	if config.FetchAMI {
		ami, err := fetchAMI(client, logger)
		if err != nil {
			return workflow.ClusterSuiteParams{}, errors.Wrap(err, "failed to fetch AMI")
		}
		createClusterReq.AMI = ami
	}

	// TODO: A way to fetch the latest AMI automatically for local development

	err := clusterdictionary.ApplyToCreateClusterRequest("SizeAlef1000", createClusterReq)
	if err != nil {
		return workflow.ClusterSuiteParams{}, errors.Wrap(err, "failed to apply cluster size")
	}

	return workflow.ClusterSuiteParams{
		UseExistingCluster: false,
		CreateRequest:      *createClusterReq,
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

// Run runs the test workflow.
func (w *Test) Run() error {
	err := workflow.RunWorkflow(w.Workflow, w.Logger)
	if err != nil {
		return errors.Wrap(err, "error running workflow")
	}
	return nil
}

func (w *Test) CleanupTest(t *testing.T) error {
	if w.Cleanup {
		err := w.InstallationSuite.Cleanup(context.Background())
		if err != nil {
			w.Logger.WithError(err).Error("Error cleaning up installation")
		}
		err = w.ClusterSuite.Cleanup(context.Background())
		if err != nil {
			w.Logger.WithError(err).Error("Error cleaning up cluster")
		}
	}

	// Always cleanup webhook
	err := w.WebhookCleanup()
	assert.NoError(t, err)
	return nil
}

func fetchAMI(cloudClient *model.Client, logger logrus.FieldLogger) (string, error) {
	clusters, err := cloudClient.GetClusters(&model.GetClustersRequest{Paging: model.AllPagesNotDeleted()})
	if err != nil {
		return "", errors.Wrap(err, "failed to get clusters to fetch AMI")
	}

	if len(clusters) == 0 {
		return "", errors.Errorf("no clusters found to fetch AMI")
	}

	if clusters[0].Provider == model.ProviderExternal {
		ami, err := fetchAMIFromAWS(logger)
		if err != nil {
			return "", errors.Wrap(err, "failed to fetch AMI from AWS")
		}
		logrus.Infof("Fetched AMI from AWS: %q", ami)
		return ami, nil
	}

	ami := clusters[0].ProvisionerMetadataKops.AMI
	logrus.Infof("Fetched AMI from existing cluster: %q", ami)

	return ami, nil
}

func fetchAMIFromAWS(logger logrus.FieldLogger) (string, error) {
	awsConfig, err := awsTools.NewAWSConfig(context.TODO())
	if err != nil {
		return "", errors.Wrap(err, "failed to build aws configuration")
	}

	awsClient, err := awsTools.NewAWSClientWithConfig(&awsConfig, logrus.New())
	if err != nil {
		return "", errors.Wrap(err, "failed to build AWS client")
	}

	ami, err := awsClient.GetAMIByTag("Name", model.AMDKopsAmiName, logger)
	if err != nil {
		return "", errors.Wrap(err, "failed to get AMI image by tag")
	}
	return ami, nil
}

func TestMain(m *testing.M) {
	// This is mainly used to send a notification when tests are finished to a mattermost webhook
	// provided with the WEBHOOOK_URL environment variable.
	state.StartTime = time.Now()
	code := m.Run()
	state.EndTime = time.Now()

	// Notify if we receive any signal
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		for sig := range c {
			fmt.Printf("caught signal: %s", sig)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	var err error
	if code != 0 {
		err = pkg.SendE2EResult(ctx, webhookFailedEmoji, webhookFailedMessage, webhookAttachmentColorError)
	} else {
		err = pkg.SendE2EResult(ctx, webhookSuccessEmoji, webhookSuccessfulMessage, webhookAttachmentColorSuccess)
	}

	if err != nil {
		fmt.Printf("error sending webhook: %s", err)
	}

	os.Exit(code)
}
