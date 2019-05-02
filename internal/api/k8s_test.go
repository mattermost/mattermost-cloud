package api_test

import (
	"github.com/mattermost/mattermost-cloud/internal/tools/k8s"
	apiv1 "k8s.io/api/core/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type mockK8sClient struct {
	outputs map[string]string
}

func newMockK8sClient() *mockK8sClient {
	return &mockK8sClient{}
}

func (m *mockK8sClient) GetPods(string) ([]apiv1.Pod, error) {
	return []apiv1.Pod{dummyPod()}, nil
}

func (m *mockK8sClient) CreateNamespace(string) (*corev1.Namespace, error) {
	return nil, nil
}

func (m *mockK8sClient) CreateFromFile(file k8s.ManifestFile) error {
	return nil
}

func (m *mockK8sClient) CreateFromFiles(file []k8s.ManifestFile) error {
	return nil
}

func (m *mockK8sClient) WaitForPodRunning(string, string, int) (apiv1.Pod, error) {
	return dummyPod(), nil
}

func dummyPod() apiv1.Pod {
	return apiv1.Pod{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Name: "dummy-pod"},
		Spec:       apiv1.PodSpec{},
		Status:     apiv1.PodStatus{},
	}
}
