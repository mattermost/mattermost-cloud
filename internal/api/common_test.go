package api_test

import "github.com/mattermost/mattermost-cloud/model"

type mockSupervisor struct {
}

func (s *mockSupervisor) Do() error {
	return nil
}

type mockProvisioner struct {
}

func (s *mockProvisioner) ExecMattermostCLI(*model.Cluster, *model.ClusterInstallation, ...string) ([]byte, error) {
	return []byte(`{"ServiceSettings":{"SiteURL":"http://test.example.com"}}`), nil
}

func sToP(s string) *string {
	return &s
}
