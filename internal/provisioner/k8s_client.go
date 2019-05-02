package provisioner

import (
	"github.com/mattermost/mattermost-cloud/internal/tools/k8s"
	log "github.com/sirupsen/logrus"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
)

type k8sFactoryFunc func(configLocation string, logger log.FieldLogger) (K8sClient, error)

// K8sClient describes the interface required by the provisioner to interact with k8s.
type K8sClient interface {
	GetPods(string) ([]apiv1.Pod, error)
	CreateNamespace(string) (*corev1.Namespace, error)
	CreateNamespaces([]string) ([]*corev1.Namespace, error)
	CreateFromFile(file k8s.ManifestFile) error
	CreateFromFiles(file []k8s.ManifestFile) error
	WaitForPodRunning(string, string, int) (apiv1.Pod, error)
}
