package k8s

import (
	"k8s.io/client-go/tools/clientcmd"
)

type KubeconfigCreds struct {
	ApiServer string
	ClusterCA []byte
	ClientCA  []byte
	ClientKey []byte
}

// ReadKubeconfigFileCreds takes kubeconfig file and load into KubeconfigCreds
func ReadKubeconfigFileCreds(kubeconfigPath string) (*KubeconfigCreds, error) {
	config, err := clientcmd.LoadFromFile(kubeconfigPath)
	if err != nil {
		return nil, err
	}

	kubeconfig := &KubeconfigCreds{}

	clusters := config.Clusters
	for _, clusterConfig := range clusters {
		kubeconfig.ApiServer = clusterConfig.Server
		kubeconfig.ClusterCA = clusterConfig.CertificateAuthorityData
	}

	authInfos := config.AuthInfos
	for _, authInfo := range authInfos {
		kubeconfig.ClientCA = authInfo.ClientCertificateData
		kubeconfig.ClientKey = authInfo.ClientKeyData
	}
	return kubeconfig, nil
}
