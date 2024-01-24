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
)

type ClusterRegister struct {
	cluster              *model.Cluster
	cloudEnvironmentName string
	gitClient            git.Client
	logger               log.FieldLogger
	tempDir              string
	clusterName          string
	clusterFile          string
	clusterFilePath      string
}

// NewClusterRegisterHandle returns a new ClusterRegister for register cluster into argocd
func NewClusterRegisterHandle(cluster *model.Cluster, gitClient git.Client, cloudEnvironmentName, tempDir string, logger log.FieldLogger) (*ClusterRegister, error) {

	clusterRegister := &ClusterRegister{
		cluster:              cluster,
		cloudEnvironmentName: cloudEnvironmentName,
		gitClient:            gitClient,
		logger:               logger,
		tempDir:              tempDir,
		clusterName:          cloudEnvironmentName + "-" + cluster.ID,
		clusterFile:          path.Join("clusters", cloudEnvironmentName, argocdClusterFileName),
		clusterFilePath:      path.Join(tempDir, "clusters", cloudEnvironmentName, argocdClusterFileName),
	}
	return clusterRegister, nil
}

func (cr *ClusterRegister) clusterRegister(s3StateStore string) error {
	logger := cr.logger.WithField("cluster", cr.cluster.ID)

	clusterCreds, err := cr.getClusterCreds(s3StateStore)
	if err != nil {
		return errors.Wrap(err, "failed to get cluster credentials")
	}

	err = cr.gitClient.Pull(logger)
	if err != nil {
		return errors.Wrap(err, "failed to pull from argocd repo")
	}

	if err = cr.updateClusterFile(clusterCreds); err != nil {
		return errors.Wrap(err, "failed to update cluster file")
	}

	commitMsg := "Adding new cluster: " + cr.cluster.ID
	if err = cr.gitClient.Commit(cr.clusterFile, commitMsg, logger); err != nil {
		return errors.Wrap(err, "failed to commit to repo")
	}

	if err = cr.gitClient.Push("feat-CLD-5708", logger); err != nil {
		return errors.Wrap(err, "failed to push to repo")
	}

	return nil
}

func (cr *ClusterRegister) updateClusterFile(clusterCreds *k8s.KubeconfigCreds) error {
	clusteFile, err := os.ReadFile(cr.clusterFilePath)
	if err != nil {
		return errors.Wrap(err, "failed to read cluster file")
	}

	clusterFile, err := argocd.ReadArgoK8sRegistrationFile(clusteFile)
	if err != nil {
		return errors.Wrap(err, "failed to load cluster registration file into argo struct")
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
		CaData:    b64.StdEncoding.EncodeToString(clusterCreds.ClusterCA),
		CertData:  b64.StdEncoding.EncodeToString(clusterCreds.ClientCA),
		KeyData:   b64.StdEncoding.EncodeToString(clusterCreds.ClientKey),
	}

	if err = argocd.UpdateK8sClusterRegistrationFile(clusterFile, newCluster, cr.clusterFilePath); err != nil {
		return errors.Wrap(err, "failed to update cluster registration file")
	}
	return nil
}

func (cr *ClusterRegister) getClusterCreds(s3StateStore string) (*k8s.KubeconfigCreds, error) {
	kopsClient, err := kops.New(s3StateStore, cr.logger)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create new kops client")
	}
	if err = kopsClient.ExportKubecfg(cr.cluster.ProvisionerMetadataKops.Name); err != nil {
		return nil, errors.Wrap(err, "failed export kube config")
	}

	clusterCreds, err := k8s.ReadKubeconfigFileCreds(kopsClient.GetKubeConfigPath())
	if err != nil {
		return nil, errors.Wrap(err, "failed read kube config file")
	}
	return clusterCreds, nil
}

func (cr *ClusterRegister) deregisterClusterFromArgocd() error {
	logger := cr.logger.WithField("cluster", cr.cluster.ID)

	clusteFile, err := os.ReadFile(cr.clusterFilePath)
	if err != nil {
		return errors.Wrap(err, "failed to read cluster file")
	}

	argoK8sFile, err := argocd.ReadArgoK8sRegistrationFile(clusteFile)
	if err != nil {
		return errors.Wrap(err, "failed to load cluster registration file into argo struct")
	}

	if err = argocd.DeleteK8sClusterFromRegistrationFile(argoK8sFile, cr.clusterName, cr.clusterFilePath); err != nil {
		return errors.Wrap(err, "failed to remove cluster from registration file")
	}

	commitMsg := "Removing cluster: " + cr.cluster.ID
	if err = cr.gitClient.Commit(cr.clusterFile, commitMsg, logger); err != nil {
		return errors.Wrap(err, "failed to commit to repo")
	}

	if err = cr.gitClient.Push("feat-CLD-5708", logger); err != nil {
		return errors.Wrap(err, "failed to push to repo")
	}

	//TODO sync application after removing cluster from argocd

	return nil
}
