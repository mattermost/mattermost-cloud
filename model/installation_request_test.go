package model_test

import (
	"bytes"
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateInstallationRequestValid(t *testing.T) {
	var testCases = []struct {
		testName     string
		requireError bool
		request      *model.CreateInstallationRequest
	}{
		{
			"defaults",
			false,
			&model.CreateInstallationRequest{
				OwnerID: "owner1",
				DNS:     "domain.com",
			},
		},
		{
			"no owner ID",
			true,
			&model.CreateInstallationRequest{
				DNS: "domain.com",
			},
		},
		{
			"no DNS",
			true,
			&model.CreateInstallationRequest{
				OwnerID: "owner1",
			},
		},
		{
			"DNS too long",
			true,
			&model.CreateInstallationRequest{
				OwnerID: "owner1",
				DNS:     "asupersupersupersupersupersupersupersupersupersupersupersuperlongname.com",
			},
		},
		{
			"invalid installation size",
			true,
			&model.CreateInstallationRequest{
				OwnerID: "owner1",
				DNS:     "domain.com",
				Size:    "jumbo",
			},
		},
		{
			"invalid affinity size",
			true,
			&model.CreateInstallationRequest{
				OwnerID:  "owner1",
				DNS:      "domain.com",
				Affinity: "solo",
			},
		},
		{
			"invalid database",
			true,
			&model.CreateInstallationRequest{
				OwnerID:  "owner1",
				DNS:      "domain.com",
				Database: "none",
			},
		},
		{
			"invalid filestore",
			true,
			&model.CreateInstallationRequest{
				OwnerID:   "owner1",
				DNS:       "domain.com",
				Filestore: "none",
			},
		},
		{
			"invalid mattermost env",
			true,
			&model.CreateInstallationRequest{
				OwnerID: "owner1",
				DNS:     "domain.com",
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: ""},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			tc.request.SetDefaults()

			if tc.requireError {
				assert.Error(t, tc.request.Validate())
			} else {
				assert.NoError(t, tc.request.Validate())
			}
		})
	}
}

func TestCreateInstallationRequestFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		request, err := model.NewCreateInstallationRequestFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.Error(t, err)
		require.Nil(t, request)
	})

	t.Run("invalid request", func(t *testing.T) {
		installation, err := model.NewCreateInstallationRequestFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, installation)
	})

	t.Run("request", func(t *testing.T) {
		request, err := model.NewCreateInstallationRequestFromReader(bytes.NewReader([]byte(`{
			"OwnerID":"owner",
			"Version":"version",
			"DNS":"dns",
			"License": "this_is_my_license",
			"MattermostEnv": {"key1": {"Value": "value1"}},
			"Affinity":"multitenant"
		}`)))
		require.NoError(t, err)

		expected := &model.CreateInstallationRequest{
			OwnerID:       "owner",
			Version:       "version",
			DNS:           "dns",
			License:       "this_is_my_license",
			MattermostEnv: model.EnvVarMap{"key1": {Value: "value1"}},
			Affinity:      "multitenant",
		}
		expected.SetDefaults()
		require.Equal(t, expected, request)
		require.NoError(t, request.Validate())
	})
}

func TestUpdateInstallationRequestValid(t *testing.T) {
	var testCases = []struct {
		testName     string
		requireError bool
		request      *model.UpdateInstallationRequest
	}{
		{
			"defaults",
			false,
			&model.UpdateInstallationRequest{
				Version: "stable",
			},
		},
		{
			"no version",
			true,
			&model.UpdateInstallationRequest{},
		},
		{
			"invalid mattermost env",
			true,
			&model.UpdateInstallationRequest{
				Version: "stable",
				MattermostEnv: model.EnvVarMap{
					"key1": {Value: ""},
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.testName, func(t *testing.T) {
			tc.request.SetDefaults()

			if tc.requireError {
				assert.Error(t, tc.request.Validate())
			} else {
				assert.NoError(t, tc.request.Validate())
			}
		})
	}
}

func TestUpdateInstallationRequestFromReader(t *testing.T) {
	t.Run("empty request", func(t *testing.T) {
		request, err := model.NewUpdateInstallationRequestFromReader(bytes.NewReader([]byte(
			``,
		)))
		require.Error(t, err)
		require.Nil(t, request)
	})

	t.Run("invalid request", func(t *testing.T) {
		installation, err := model.NewUpdateInstallationRequestFromReader(bytes.NewReader([]byte(
			`{test`,
		)))
		require.Error(t, err)
		require.Nil(t, installation)
	})

	t.Run("request", func(t *testing.T) {
		request, err := model.NewUpdateInstallationRequestFromReader(bytes.NewReader([]byte(`{
			"Version":"version",
			"License": "this_is_my_license",
			"MattermostEnv": {"key1": {"Value": "value1"}}
		}`)))
		require.NoError(t, err)

		expected := &model.UpdateInstallationRequest{
			Version:       "version",
			License:       "this_is_my_license",
			MattermostEnv: model.EnvVarMap{"key1": {Value: "value1"}},
		}
		expected.SetDefaults()
		require.Equal(t, expected, request)
		require.NoError(t, request.Validate())
	})
}
