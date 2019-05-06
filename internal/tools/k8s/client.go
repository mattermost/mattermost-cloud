package k8s

import (
	log "github.com/sirupsen/logrus"

	"github.com/kubernetes/client-go/tools/clientcmd"
	mmclient "github.com/mattermost/mattermost-cloud/internal/tools/k8s/pkg/clientset/versioned"
	apixclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// KubeClient interfaces with a Kubernetes cluster in the same way kubectl would.
type KubeClient struct {
	config              *rest.Config
	Clientset           kubernetes.Interface
	ApixClientset       apixclient.Interface
	MattermostClientset mmclient.Interface
	logger              log.FieldLogger
}

// New returns a new KubeClient for accessing the kubernetes API.
func New(configLocation string, logger log.FieldLogger) (*KubeClient, error) {
	config, err := clientcmd.BuildConfigFromFlags("", configLocation)
	if err != nil {
		return &KubeClient{}, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return &KubeClient{}, err
	}

	apixClientset, err := apixclient.NewForConfig(config)
	if err != nil {
		return &KubeClient{}, err
	}

	mattermostClientset, err := mmclient.NewForConfig(config)
	if err != nil {
		return &KubeClient{}, err
	}

	return &KubeClient{
			config:              config,
			Clientset:           clientset,
			MattermostClientset: mattermostClientset,
			ApixClientset:       apixClientset,
			logger:              logger,
		},
		nil
}

func (kc *KubeClient) getKubeConfigClientset() (*kubernetes.Clientset, error) {
	return kubernetes.NewForConfig(kc.config)
}
