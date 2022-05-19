// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCustomColumnsTable(t *testing.T) {
	columnsExpression := []string{"ID:.ID", "Owner:{.OwnerID}", "DNS:.DNSRecords[0].DomainName", "State:State", "FirstAnnotation:{Annotations[0].Name}", "Smell:.Smell"}

	data := []interface{}{
		model.InstallationDTO{
			Installation: &model.Installation{
				ID:      "installation-1",
				OwnerID: "unit-test",
				State:   model.InstallationStateStable,
			},
			Annotations: []*model.Annotation{
				{Name: "test", ID: "annotation-1"},
			},
			DNSRecords: []*model.InstallationDNS{
				{DomainName: "unit-test.mattermost.com"},
			},
		},
		model.InstallationDTO{
			Installation: &model.Installation{
				ID:      "installation-2",
				OwnerID: "unit-test",
				State:   model.InstallationStateCreationInProgress,
			},
			Annotations: []*model.Annotation{
				{Name: "test-123", ID: "annotation-2"},
			},
			DNSRecords: []*model.InstallationDNS{
				{DomainName: "unit-test2.mattermost.com"},
			},
		},
		model.InstallationDTO{
			Installation: &model.Installation{
				ID:    "installation-3",
				State: model.InstallationStateDeleted,
			},
			Annotations: []*model.Annotation{
				{Name: "test-123", ID: "annotation-2"},
				{Name: "test", ID: "annotation-1"},
			},
			DNSRecords: []*model.InstallationDNS{
				{DomainName: "unit-test3.mattermost.com"},
			},
		},
	}

	keys, vals, err := prepareTableData(columnsExpression, data)
	require.NoError(t, err)

	expectedVals := [][]string{
		{"installation-1", "unit-test", "unit-test.mattermost.com", "stable", "test", "<none>"},
		{"installation-2", "unit-test", "unit-test2.mattermost.com", "creation-in-progress", "test-123", "<none>"},
		{"installation-3", "", "unit-test3.mattermost.com", "deleted", "test-123", "<none>"},
	}

	assert.Equal(t, []string{"ID", "Owner", "DNS", "State", "FirstAnnotation", "Smell"}, keys)
	assert.Equal(t, expectedVals, vals)
}
