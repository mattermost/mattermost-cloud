// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/util"
	"github.com/stretchr/testify/require"
)

func TestInstallationDTOFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		installationDTO, err := DTOFromReader[InstallationDTO](bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, &InstallationDTO{}, installationDTO)
	})

	t.Run("invalid request", func(t *testing.T) {
		installationDTO, err := DTOFromReader[InstallationDTO](bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, installationDTO)
	})

	t.Run("request", func(t *testing.T) {
		installationDTO, err := DTOFromReader[InstallationDTO](bytes.NewReader([]byte(`{
			"ID":"id",
			"OwnerID":"owner",
			"GroupID":"group_id",
			"Version":"version",
			"DNS":"dns",
			"License": "this_is_my_license",
			"MattermostEnv": {"key1": {"Value": "value1"}},
			"Affinity":"affinity",
			"State":"state",
			"CreateAt":10,
			"DeleteAt":20,
			"LockAcquiredAt":0,
			"Annotations": [
				{"ID": "abc", "Name": "efg"}
			]
		}`)))
		require.NoError(t, err)
		require.Equal(t, &InstallationDTO{
			Installation: &Installation{
				ID:             "id",
				OwnerID:        "owner",
				GroupID:        util.SToP("group_id"),
				Version:        "version",
				License:        "this_is_my_license",
				MattermostEnv:  EnvVarMap{"key1": {Value: "value1"}},
				Affinity:       "affinity",
				State:          "state",
				CreateAt:       10,
				DeleteAt:       20,
				LockAcquiredBy: nil,
				LockAcquiredAt: int64(0),
			},
			DNS:         "dns",
			Annotations: []*Annotation{{ID: "abc", Name: "efg"}},
		}, installationDTO)
	})
}

func TestInstallationDTOsFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		installationDTOs, err := DTOsFromReader[InstallationDTO](bytes.NewReader([]byte(
			``,
		)))
		require.NoError(t, err)
		require.Equal(t, []*InstallationDTO{}, installationDTOs)
	})

	t.Run("invalid request", func(t *testing.T) {
		installationDTOs, err := DTOsFromReader[InstallationDTO](bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, installationDTOs)
	})

	t.Run("request", func(t *testing.T) {
		installationDTOs, err := DTOsFromReader[InstallationDTO](bytes.NewReader([]byte(`[
			{
				"ID":"id1",
				"OwnerID":"owner1",
				"GroupID":"group_id1",
				"Version":"version1",
				"DNS":"dns1",
				"MattermostEnv": {"key1": {"Value": "value1"}},
				"Affinity":"affinity1",
				"State":"state1",
				"CreateAt":10,
				"DeleteAt":20,
				"LockAcquiredAt":0,
				"Annotations": [
					{"ID": "abc", "Name": "efg"}
				]
			},
			{
				"ID":"id2",
				"OwnerID":"owner2",
				"GroupID":"group_id2",
				"Version":"version2",
				"DNS":"dns2",
				"License": "this_is_my_license",
				"MattermostEnv": {"key2": {"Value": "value2"}},
				"Affinity":"affinity2",
				"State":"state2",
				"CreateAt":30,
				"DeleteAt":40,
				"LockAcquiredBy": "tester",
				"LockAcquiredAt":50
			}
		]`)))
		require.NoError(t, err)
		require.Equal(t, []*InstallationDTO{
			{
				Installation: &Installation{
					ID:             "id1",
					OwnerID:        "owner1",
					GroupID:        util.SToP("group_id1"),
					Version:        "version1",
					MattermostEnv:  EnvVarMap{"key1": {Value: "value1"}},
					Affinity:       "affinity1",
					State:          "state1",
					CreateAt:       10,
					DeleteAt:       20,
					LockAcquiredBy: nil,
					LockAcquiredAt: 0,
				},
				DNS:         "dns1",
				Annotations: []*Annotation{{ID: "abc", Name: "efg"}},
			},
			{
				Installation: &Installation{
					ID:             "id2",
					OwnerID:        "owner2",
					GroupID:        util.SToP("group_id2"),
					Version:        "version2",
					License:        "this_is_my_license",
					MattermostEnv:  EnvVarMap{"key2": {Value: "value2"}},
					Affinity:       "affinity2",
					State:          "state2",
					CreateAt:       30,
					DeleteAt:       40,
					LockAcquiredBy: util.SToP("tester"),
					LockAcquiredAt: 50,
				},
				DNS: "dns2",
			},
		}, installationDTOs)
	})
}
