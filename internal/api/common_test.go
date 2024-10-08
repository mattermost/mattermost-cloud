// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api_test

import (
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
)

type mockSupervisor struct{}

func (s *mockSupervisor) Do() error {
	return nil
}

type mockDNSProvider struct{}

func (s *mockDNSProvider) DeleteDNSRecords(customerDNSName []string, logger log.FieldLogger) error {
	return nil
}

type mockMetrics struct{}

func (m *mockMetrics) IncrementAPIRequest() {}

func (m *mockMetrics) ObserveAPIEndpointDuration(handler, method string, statusCode int, elapsed float64) {
}

type mockProvisioner struct {
	Output       []byte
	DebugData    model.ClusterInstallationDebugData
	ExecError    error
	CommandError error
}

func (s *mockProvisioner) ProvisionerType() string {
	return "kops"
}

func (s *mockProvisioner) GetClusterInstallationStatus(cluster *model.Cluster, clusterInstallation *model.ClusterInstallation) (*model.ClusterInstallationStatus, error) {
	return &model.ClusterInstallationStatus{}, nil
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

func (s *mockProvisioner) ExecClusterInstallationPPROF(*model.Cluster, *model.ClusterInstallation) (model.ClusterInstallationDebugData, error, error) {
	return s.DebugData, s.ExecError, s.CommandError
}

func (s *mockProvisioner) GetClusterResources(*model.Cluster, bool, log.FieldLogger) (*k8s.ClusterResources, error) {
	return nil, nil
}
