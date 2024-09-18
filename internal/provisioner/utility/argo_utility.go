package utility

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

	// Create the cluster output directory for the utility
	clusterOutputDir := filepath.Join(tempDir, "apps", awsClient.GetCloudEnvironmentName(), "helm-values", clusterID)
	err = os.MkdirAll(clusterOutputDir, os.ModePerm)
	if err != nil && !os.IsExist(err) {
		return errors.Wrap(err, "failed to create cluster output directory for utility")
	}

	inputFilePath := filepath.Join(tempDir, "apps/custom-values-template", utilityName+"-custom-values.yaml-template")
	outputFilePath := filepath.Join(clusterOutputDir, utilityName+"-custom-values.yaml")

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

	privateCertificate, err := awsClient.GetCertificateSummaryByTag(aws.DefaultInstallPrivateCertificatesTagKey, aws.DefaultInstallPrivateCertificatesTagValue, logger)
	if err != nil {
		return errors.Wrap(err, "failed to retrive the AWS Private ACM")
	}

	if privateCertificate.ARN == nil {
		return errors.New("retrieved certificate does not have ARN")
	}

	privateZoneDomainName, err := awsClient.GetPrivateZoneDomainName(logger)
	if err != nil {
		return errors.Wrap(err, "failed to get private zone domain name")
	}

	replacements := map[string]string{
		"<VPC_ID>":                  vpc,
		"<CERTIFICATE_ARN>":         *certificate.ARN,
		"<PRIVATE_CERTIFICATE_ARN>": *privateCertificate.ARN,
		"<CLUSTER_ID>":              clusterID,
		"<ENV>":                     awsClient.GetCloudEnvironmentName(),
		"<IP_RANGE>":                strings.Join(allowCIDRRangeList, ","),
		"<PRIVATE_DOMAIN>":          privateZoneDomainName,
		// Add more replacements as needed
	}

	// Skip substitution if the output file already exists
	if _, err = os.Stat(outputFilePath); err == nil {
		logger.Debugf("Output file already exists, skipping substitution for %s", outputFilePath)
		return nil
	}

	// Perform substitution
	_, err = os.Stat(inputFilePath)
	if os.IsNotExist(err) {
		return errors.Wrap(err, "custom values template file does not exist")
	}

	err = substituteValues(inputFilePath, outputFilePath, replacements, logger)
	if err != nil {
		return errors.Wrap(err, "failed to substitute values")
	}

	commitMsg := "Adding: utility:" + utilityName + " to cluster: " + clusterID
	if err = gitClient.Commit(tempDir+"/apps", commitMsg, logger); err != nil {
		return errors.Wrap(err, "failed to commit to repo")
	}

	if err = gitClient.Push(logger); err != nil {
		return errors.Wrap(err, "failed to push to repo")
	}

	appName := utilityName + "-sre-" + awsClient.GetCloudEnvironmentName() + "-" + clusterID

	err = argocdClient.WaitForAppHealthy(appName)
	if err != nil {
		return errors.Wrap(err, "failed to wait for application to be healthy")
	}

	logger.WithField("Utility:", utilityName).Info("Deployed utility successfully.")

	return nil
}

func (group utilityGroup) RemoveUtilityFromArgocd(gitClient git.Client) error {

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

	applicationFile := filepath.Join(group.tempDir, "apps", group.awsClient.GetCloudEnvironmentName(), ArgocdAppsFile)

	// Git pull to get the latest state before deleting the cluster
	err = gitClient.Pull(group.logger)
	if err != nil {
		return errors.Wrap(err, "failed to pull from argocd repo")
	}

	commitMsg := "Removing Utilities: " + group.cluster.ID
	if err = gitClient.Commit(applicationFile, commitMsg, group.logger); err != nil {
		return errors.Wrap(err, "failed to commit to repo")
	}

	if err = gitClient.Push(group.logger); err != nil {
		return errors.Wrap(err, "failed to push to repo")
	}

	return nil

}

func substituteValues(inputFilePath, outputFilePath string, replacements map[string]string, logger log.FieldLogger) error {
	inputFile, err := os.Open(inputFilePath)
	if err != nil {
		return errors.Wrap(err, "failed to open input file for argo utility substitution")
	}
	defer inputFile.Close()

	outputFile, err := os.Create(outputFilePath)
	if err != nil {
		return errors.Wrap(err, "failed to create output file for argo utility substitution")
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
