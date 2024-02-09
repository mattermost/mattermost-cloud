package utility

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/mattermost/mattermost-cloud/internal/tools/argocd"
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/mattermost/mattermost-cloud/internal/tools/git"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

const (
	ArgocdAppsFile = "/application-values.yaml"
)

func ProvisionUtilityArgocd(utilityName, tempDir, clusterID string, allowCIDRRangeList []string, awsClient aws.AWS,
	gitClient git.Client, argocdClient argocd.Client, logger log.FieldLogger) error {
	// Pull the latest changes from the repo
	err := gitClient.Pull(logger)
	if err != nil {
		return errors.Wrap(err, "failed to pull from repo")
	}

	//TODO: Skip provision utility if it is already provisioned

	appsFile, err := os.ReadFile(tempDir + "/apps/" + awsClient.GetCloudEnvironmentName() + ArgocdAppsFile)
	if err != nil {
		return errors.Wrap(err, "failed to read cluster file")
	}
	argoAppFile, err := argocd.ReadArgoApplicationFile(appsFile)
	if err != nil {
		return errors.Wrap(err, "failed to read argo application file")
	}
	argocd.AddClusterIDLabel(argoAppFile, utilityName, clusterID, logger)
	modifiedYAML, err := yaml.Marshal(argoAppFile)
	if err != nil {
		return errors.Wrap(err, "failed to marshal argo application file")
	}
	if err = os.WriteFile(tempDir+"/apps/"+awsClient.GetCloudEnvironmentName()+ArgocdAppsFile, modifiedYAML, 0644); err != nil {
		return errors.Wrap(err, "failed to write argo application file")
	}

	inputFilePath := tempDir + "/apps/custom-values-template/" + utilityName + "-custom-values.yaml-template"
	outputFilePath := tempDir + "/apps/" + awsClient.GetCloudEnvironmentName() + "/helm-values/" + clusterID + "/" + utilityName + "-custom-values.yaml"

	vpc, err := awsClient.GetClaimedVPC(clusterID, logger)
	if err != nil {
		return errors.Wrapf(err, "failed to perform VPC lookup for cluster %s", clusterID)
	}

	if vpc == "" {
		return errors.New("no VPC found for cluster")
	}

	certificate, err := awsClient.GetCertificateSummaryByTag(aws.DefaultInstallCertificatesTagKey, aws.DefaultInstallCertificatesTagValue, logger)
	if err != nil {
		return errors.Wrap(err, "failed to retrive the AWS ACM")
	}

	if certificate.ARN == nil {
		return errors.New("retrieved certificate does not have ARN")
	}

	privateZoneDomainName, err := awsClient.GetPrivateZoneDomainName(logger)
	if err != nil {
		return errors.Wrap(err, "failed to get private zone domain name")
	}

	replacements := map[string]string{
		"<VPC_ID>":         vpc,
		"<CERTFICATE_ARN>": *certificate.ARN,
		"<CLUSTER_ID>":     clusterID,
		"<ENV>":            awsClient.GetCloudEnvironmentName(),
		"<IP_RANGE>":       strings.Join(allowCIDRRangeList, ","),
		"<PRIVATE_DOMAIN>": privateZoneDomainName,
		// Add more replacements as needed
	}

	// Perform substitution
	_, err = os.Stat(outputFilePath)
	if os.IsNotExist(err) {
		err = substituteValues(inputFilePath, outputFilePath, replacements)
		if err != nil {
			return errors.Wrap(err, "failed to substitute values")
		} else {
			logger.WithField("Check the output file:", outputFilePath).Info("Substitution successful.")
		}
	}

	commitMsg := "Adding: utility:" + utilityName + " to cluster: " + clusterID
	if err = gitClient.Commit(tempDir+"/apps", commitMsg, logger); err != nil {
		return errors.Wrap(err, "failed to commit to repo")
	}

	if err = gitClient.Push("feat-CLD-5708", logger); err != nil {
		return errors.Wrap(err, "failed to push to repo")
	}

	appName := utilityName + "-sre-" + awsClient.GetCloudEnvironmentName() + "-" + clusterID
	gitopsAppName := "gitops-sre-" + awsClient.GetCloudEnvironmentName()
	app, err := argocdClient.SyncApplication(gitopsAppName)
	if err != nil {
		return errors.Wrap(err, "failed to sync application")
	}

	var wg sync.WaitGroup
	timeout := time.Second * 300

	wg.Add(1)
	go argocdClient.WaitForAppHealthy(appName, &wg, timeout) //TODO: return error
	wg.Wait()

	logger.WithField("app:", app.Name).Info("Deployed utility successfully.")

	return nil
}

func (group utilityGroup) RemoveUtilityFromArgocd() error {

	appsFile, err := os.ReadFile(group.tempDir + "/apps/" + group.awsClient.GetCloudEnvironmentName() + ArgocdAppsFile)
	if err != nil {
		return errors.Wrap(err, "failed to read cluster file")
	}
	argoAppFile, err := argocd.ReadArgoApplicationFile(appsFile)
	if err != nil {
		return errors.Wrap(err, "failed to read argo application file")
	}

	for _, utility := range group.utilities {
		argocd.RemoveClusterIDLabel(argoAppFile, utility.Name(), group.cluster.ID, group.logger)

		var modifiedYAML []byte
		modifiedYAML, err = yaml.Marshal(argoAppFile)
		if err != nil {
			return errors.Wrap(err, "failed to marshal argo application file")
		}
		if err = os.WriteFile(group.tempDir+"/apps/"+group.awsClient.GetCloudEnvironmentName()+ArgocdAppsFile, modifiedYAML, 0644); err != nil {
			return errors.Wrap(err, "failed to write argo application file")
		}
	}

	if err = os.RemoveAll(group.tempDir + "/apps/" + group.awsClient.GetCloudEnvironmentName() + "/helm-values/" + group.cluster.ID); err != nil {
		return errors.Wrap(err, "failed to remove helm values directory")
	}

	return nil

}

func substituteValues(inputFilePath, outputFilePath string, replacements map[string]string) error {
	inputFile, err := os.Open(inputFilePath)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	scanner := bufio.NewScanner(inputFile)

	for scanner.Scan() {
		line := scanner.Text()

		for placeholder, replacement := range replacements {
			line = strings.ReplaceAll(line, placeholder, replacement)
		}

		_, err := fmt.Fprintln(outputFile, line)
		if err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}
