// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package api_test

import (
	"github.com/mattermost/mattermost-cloud/k8s"
	"github.com/mattermost/mattermost-cloud/model"
)

type mockSupervisor struct {
}

func (s *mockSupervisor) Do() error {
	return nil
}

type mockProvisioner struct {
	Output       []byte
	CommandError error
}

func (s *mockProvisioner) ExecMattermostCLI(*model.Cluster, *model.ClusterInstallation, ...string) ([]byte, error) {
	if len(s.Output) == 0 {
		s.Output = []byte(`{"ServiceSettings":{"SiteURL":"http://test.example.com"}}`)
	}

	return s.Output, s.CommandError
}

func (s *mockProvisioner) GetClusterResources(*model.Cluster, bool) (*k8s.ClusterResources, error) {
	return nil, nil
}

func sToP(s string) *string {
	return &s
}
