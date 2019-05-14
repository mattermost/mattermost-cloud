package k8s

import (
	"github.com/sirupsen/logrus"

	mmfake "github.com/mattermost/mattermost-operator/pkg/client/clientset/versioned/fake"
	apixfake "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset/fake"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func newTestKubeClient() *KubeClient {
	return &KubeClient{
		config:              &rest.Config{},
		Clientset:           fake.NewSimpleClientset(),
		ApixClientset:       apixfake.NewSimpleClientset(),
		MattermostClientset: mmfake.NewSimpleClientset(),
		logger:              logrus.New(),
	}
}
