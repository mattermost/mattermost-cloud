package provisioner

import (
	"os"
	"path"

	b64 "encoding/base64"

	"github.com/mattermost/mattermost-cloud/internal/tools/argocd"
	"github.com/mattermost/mattermost-cloud/internal/tools/git"
	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	argocdClusterFileName = "cluster-values.yaml"
	gitOpsRepoPath        = "/cloud-sre/kubernetes-workloads/gitops-sre.git"
)

type ClusterRegister struct {
	cluster              *model.Cluster
	cloudEnvironmentName string
	gitClient            git.Client
	logger               log.FieldLogger
	tempDir              string
}

// NewClusterRegisterHandle returns a new ClusterRegister for register cluster into argocd
func NewClusterRegisterHandle(cluster *model.Cluster, cloudEnvironmentName string, logger log.FieldLogger) (*ClusterRegister, error) {
	gitlabOAuthToken := os.Getenv(model.GitlabOAuthTokenKey)
	if len(gitlabOAuthToken) == 0 {
		return nil, errors.Errorf("The %s env was empty; unable to register cluster into argocd", model.GitlabOAuthTokenKey)
	}

	tempDir, err := os.MkdirTemp("", "cluster-register-")
	if err != nil {
		return nil, errors.Wrap(err, "failed to create temporary directory")
	}

	gitOpsRepoURL := model.GetGitopsRepoURL()
	argocdRepoURL := gitOpsRepoURL + gitOpsRepoPath
	gitClient, err := git.NewGitClient(gitlabOAuthToken, tempDir, argocdRepoURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new git client")
	}

	clusterRegister := &ClusterRegister{
		cluster:              cluster,
		cloudEnvironmentName: cloudEnvironmentName,
		gitClient:            gitClient,
		logger:               logger,
		tempDir:              tempDir,
	}

	return clusterRegister, nil
}

func (cr *ClusterRegister) clusterRegister(s3StateStore, clusterName string) error {
	logger := cr.logger.WithField("cluster", cr.cluster.ID)

	clusterCreds, err := cr.getClusterCreds(clusterName, s3StateStore)
	if err != nil {
		return errors.Wrap(err, "failed to get cluster credentials")
	}

	err = cr.gitClient.Checkout("main", logger)
	if err != nil {
		return errors.Wrap(err, "failed to checkout repo")
	}

	clusterFile := path.Join("clusters", cr.cloudEnvironmentName, argocdClusterFileName)
	clusterFilePath := path.Join(cr.tempDir, clusterFile)
	err = cr.updateClusterFile(clusterFilePath, clusterCreds)
	if err != nil {
		return errors.Wrap(err, "Error updating cluster file")
	}

	commitMsg := "Adding new cluster: " + cr.cluster.ID
	err = cr.gitClient.Commit(clusterFile, commitMsg, "Provisioner", logger)
	if err != nil {
		return errors.Wrap(err, "failed to commit to repo")
	}

	err = cr.gitClient.Push(logger)
	if err != nil {
		return errors.Wrap(err, "Failed to push to repo")
	}

	defer cr.gitClient.Close(clusterFilePath, logger)

	return nil
}

func (cr *ClusterRegister) updateClusterFile(clusterFilePath string, clusterCreds *k8s.KubeconfigCreds) error {
	clusteFile, err := os.ReadFile(clusterFilePath)
	if err != nil {
		return errors.Wrap(err, "failed to read cluster file")
	}

	var argo argocd.Argock8sRegister
	clusterFile, err := argo.ReadArgoK8sRegistrationFile(clusteFile)
	if err != nil {
		return errors.Wrap(err, "failed to read cluster registration file")
	}

	newClusterLabels := argocd.ArgocdClusterLabels{
		ClusterTypes: cr.cluster.UtilityMetadata.ArgocdClusterRegister.ClusterType,
		ClusterID:    cr.cluster.ID,
	}

	newCluster := argocd.ArgocdClusterRegisterParameters{
		Name:      cr.cloudEnvironmentName + "-" + cr.cluster.ID,
		Type:      cr.cluster.Provisioner,
		Labels:    newClusterLabels,
		APIServer: clusterCreds.ApiServer,
		CaData:    b64.StdEncoding.EncodeToString([]byte(clusterCreds.ClusterCA)),
		CertData:  b64.StdEncoding.EncodeToString([]byte(clusterCreds.ClientCA)),
		KeyData:   b64.StdEncoding.EncodeToString([]byte(clusterCreds.ClientKey)),
	}

	err = argo.UpdateK8sClusterRegistrationFile(clusterFile, newCluster, clusterFilePath)
	if err != nil {
		return errors.Wrap(err, "failed to update cluster registration file")
	}

	return nil
}

func (cr *ClusterRegister) getClusterCreds(clusterName, s3StateStore string) (*k8s.KubeconfigCreds, error) {
	kopsClient, err := kops.New(s3StateStore, cr.logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new kops client")
	}
	if err = kopsClient.ExportKubecfg(clusterName, "876000h"); err != nil {
		return nil, errors.Wrap(err, "failed export kube config")
	}

	clusterCreds, err := k8s.ReadKubeconfigFileCreds(kopsClient.GetKubeConfigPath())
	if err != nil {
		return nil, errors.Wrap(err, "failed read kube config file")
	}
	return clusterCreds, nil
}
