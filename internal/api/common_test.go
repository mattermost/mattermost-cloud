// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api_test

import (
	"github.com/mattermost/mattermost-cloud/internal/api"
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

type mockSupervisor struct{}

func (s *mockSupervisor) Do() error {
	return nil
}

type mockMetrics struct{}

func (m *mockMetrics) IncrementAPIRequest() {}

func (m *mockMetrics) ObserveAPIEndpointDuration(handler, method string, statusCode int, elapsed float64) {
}

type mockProvisionerOption struct {
	mock *mockProvisioner
}

func (p *mockProvisionerOption) GetProvisioner(provisioner string) api.Provisioner {
	if p.mock == nil {
		p.mock = &mockProvisioner{}
	}
	return p.mock
}

type mockProvisioner struct {
	Output       []byte
	ExecError    error
	CommandError error
}

func (s *mockProvisioner) ProvisionerType() string {
	return "kops"
}

func (s *mockProvisioner) ExecClusterInstallationCLI(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation, args ...string) ([]byte, error, error) {
	if len(s.Output) == 0 {
		s.Output = []byte(`{"ServiceSettings":{"SiteURL":"http://test.example.com"}}`)
	}

	return s.Output, s.ExecError, s.CommandError
}

func (s *mockProvisioner) ExecMMCTL(*model.Cluster, *model.ClusterInstallation, ...string) ([]byte, error) {
	if len(s.Output) == 0 {
		s.Output = []byte(`{"ServiceSettings":{"SiteURL":"http://test.example.com"}}`)
	}

	return s.Output, s.CommandError
}

func (s *mockProvisioner) ExecMattermostCLI(*model.Cluster, *model.ClusterInstallation, ...string) ([]byte, error) {
	if len(s.Output) == 0 {
		s.Output = []byte(`{"ServiceSettings":{"SiteURL":"http://test.example.com"}}`)
	}

	return s.Output, s.CommandError
}

func (s *mockProvisioner) GetClusterResources(*model.Cluster, bool, log.FieldLogger) (*k8s.ClusterResources, error) {
	return nil, nil
}

func sToP(s string) *string {
	return &s
}

func iToP(i int64) *int64 {
	return &i
}
