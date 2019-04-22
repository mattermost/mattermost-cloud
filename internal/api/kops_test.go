package api_test

import (
	"io/ioutil"

	"github.com/mattermost/mattermost-cloud/internal/tools/kops"
)

type mockKopsCmd struct {
	outputDirectory string
}

func newMockKopsCmd() (*mockKopsCmd, error) {
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		return nil, err
	}

	return &mockKopsCmd{
		outputDirectory: dir,
	}, nil
}

func (m *mockKopsCmd) CreateCluster(string, string, kops.ClusterSize, []string) error {
	return nil
}

func (m *mockKopsCmd) GetCluster(string) (string, error) {
	return "", nil
}

func (m *mockKopsCmd) UpdateCluster(string) error {
	return nil
}

func (m *mockKopsCmd) UpgradeCluster(string) error {
	return nil
}

func (m *mockKopsCmd) DeleteCluster(string) error {
	return nil
}

func (m *mockKopsCmd) RollingUpdateCluster(string) error {
	return nil
}

func (m *mockKopsCmd) WaitForKubernetesReadiness(string, int) error {
	return nil
}

func (m *mockKopsCmd) ValidateCluster(string, bool) error {
	return nil
}

func (m *mockKopsCmd) GetOutputDirectory() string {
	return m.outputDirectory
}

func (m *mockKopsCmd) Close() error {
	return nil
}
