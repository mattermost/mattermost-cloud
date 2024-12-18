// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/util"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/assert"
)

func TestValidateVolumeMap(t *testing.T) {
	var testCases = []struct {
		description string
		config      *model.VolumeMap
		valid       bool
	}{
		{
			"valid",
			&model.VolumeMap{
				"test1": model.Volume{Type: model.VolumeTypeSecret, MountPath: "/test1", ReadOnly: true},
				"test2": model.Volume{Type: model.VolumeTypeSecret, MountPath: "/test2", ReadOnly: true},
			},
			true,
		},
		{
			"no mount path",
			&model.VolumeMap{
				"test1": model.Volume{Type: model.VolumeTypeSecret, ReadOnly: true},
				"test2": model.Volume{Type: model.VolumeTypeSecret, MountPath: "/test2", ReadOnly: true},
			},
			false,
		},
		{
			"invalid volume type",
			&model.VolumeMap{
				"test1": model.Volume{Type: "invalid", MountPath: "/test1", ReadOnly: true},
				"test2": model.Volume{Type: model.VolumeTypeSecret, MountPath: "/test2", ReadOnly: true},
			},
			false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.description, func(t *testing.T) {
			if tc.valid {
				assert.NoError(t, tc.config.Validate())
			} else {
				assert.Error(t, tc.config.Validate())
			}
		})
	}
}

func TestVolumeMapAdd(t *testing.T) {
	var sizeTests = []struct {
		name        string
		original    *model.VolumeMap
		request     *model.CreateInstallationVolumeRequest
		expected    *model.VolumeMap
		expectError bool
	}{
		{
			"new volumes",
			&model.VolumeMap{},
			&model.CreateInstallationVolumeRequest{
				Name: "test",
				Data: map[string][]byte{"testfile": []byte("testdata")},
				Volume: &model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/test",
					ReadOnly:  false,
				},
			},
			&model.VolumeMap{
				"test": model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/test",
					ReadOnly:  false,
				},
			},
			false,
		},
		{
			"add second volume",
			&model.VolumeMap{
				"test1": model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/test1",
					ReadOnly:  false,
				},
			},
			&model.CreateInstallationVolumeRequest{
				Name: "test2",
				Data: map[string][]byte{"testfile": []byte("testdata")},
				Volume: &model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/test2",
					ReadOnly:  false,
				},
			},
			&model.VolumeMap{
				"test1": model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/test1",
					ReadOnly:  false,
				},
				"test2": model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/test2",
					ReadOnly:  false,
				},
			},
			false,
		},
		{
			"conflicting mount points",
			&model.VolumeMap{
				"test1": model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/test1",
					ReadOnly:  false,
				},
			},
			&model.CreateInstallationVolumeRequest{
				Name: "test2",
				Data: map[string][]byte{"testfile": []byte("testdata")},
				Volume: &model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/test1",
					ReadOnly:  false,
				},
			},
			&model.VolumeMap{
				"test1": model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/test1",
					ReadOnly:  false,
				},
			},
			true,
		},
		{
			"invalid name",
			&model.VolumeMap{},
			&model.CreateInstallationVolumeRequest{
				Name: "%*#$&%&#($*&%)",
				Data: map[string][]byte{"testfile": []byte("testdata")},
				Volume: &model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/test",
					ReadOnly:  false,
				},
			},
			&model.VolumeMap{},
			true,
		},
	}

	for _, tt := range sizeTests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.request != nil {
				assert.NoError(t, tt.request.Validate())
			}
			err := tt.original.Add(tt.request)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.original, tt.expected)
		})
	}
}

func TestVolumeMapUpdate(t *testing.T) {
	var sizeTests = []struct {
		name        string
		original    *model.VolumeMap
		volumeName  string
		request     *model.PatchInstallationVolumeRequest
		expected    *model.VolumeMap
		expectError bool
	}{
		{
			"update everything",
			&model.VolumeMap{
				"test": model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/old",
					ReadOnly:  false,
				},
			},
			"test",
			&model.PatchInstallationVolumeRequest{
				MountPath: util.SToP("/new"),
				ReadOnly:  util.BToP(true),
			},
			&model.VolumeMap{
				"test": model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/new",
					ReadOnly:  true,
				},
			},
			false,
		},
		{
			"update mount path only",
			&model.VolumeMap{
				"test": model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/old",
					ReadOnly:  false,
				},
			},
			"test",
			&model.PatchInstallationVolumeRequest{
				MountPath: util.SToP("/new"),
			},
			&model.VolumeMap{
				"test": model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/new",
					ReadOnly:  false,
				},
			},
			false,
		},
		{
			"volume doesn't exist",
			&model.VolumeMap{
				"test": model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/old",
					ReadOnly:  false,
				},
			},
			"invalid",
			&model.PatchInstallationVolumeRequest{
				MountPath: util.SToP("/new"),
			},
			&model.VolumeMap{
				"test": model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/old",
					ReadOnly:  false,
				},
			},
			true,
		},
		{
			"conflicting mount points",
			&model.VolumeMap{
				"test1": model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/test1",
					ReadOnly:  false,
				},
				"test2": model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/test2",
					ReadOnly:  false,
				},
			},
			"test2",
			&model.PatchInstallationVolumeRequest{
				MountPath: util.SToP("/test1"),
			},
			&model.VolumeMap{
				"test1": model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/test1",
					ReadOnly:  false,
				},
				"test2": model.Volume{
					Type:      model.VolumeTypeSecret,
					MountPath: "/test2",
					ReadOnly:  false,
				},
			},
			true,
		},
	}

	for _, tt := range sizeTests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.request != nil {
				assert.NoError(t, tt.request.Validate())
			}
			_, err := tt.original.Patch(tt.request, tt.volumeName)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.original, tt.expected)
		})
	}
}
