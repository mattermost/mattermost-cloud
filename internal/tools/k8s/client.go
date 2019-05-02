package k8s

import (
	"github.com/kubernetes/client-go/tools/clientcmd"
	log "github.com/sirupsen/logrus"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// KubeClient interfaces with a Kubernetes cluster in the same way kubectl would.
type KubeClient struct {
	config *rest.Config
	logger log.FieldLogger
}

// New returns a new KubeClient for accessing the kubernetes API.
func New(configLocation string, logger log.FieldLogger) (*KubeClient, error) {
	config, err := clientcmd.BuildConfigFromFlags("", configLocation)
	if err != nil {
		return &KubeClient{}, err
	}

	return &KubeClient{
			config: config,
			logger: logger,
		},
		nil
}

func (kc *KubeClient) getKubeConfigClientset() (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(kc.config)
}
