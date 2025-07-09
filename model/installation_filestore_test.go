// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model_test

import (
	"io"
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestInternalFilestore(t *testing.T) {
	var testCases = []struct {
		filestoreType  string
		expectInternal bool
	}{
		{"", false},
		{"unknown", false},
		{model.InstallationFilestoreMinioOperator, true},
		{model.InstallationFilestoreAwsS3, false},
		{model.InstallationFilestoreLocalEphemeral, true},
	}

	for _, tc := range testCases {
		t.Run(tc.filestoreType, func(t *testing.T) {
			installation := &model.Installation{
				Filestore: tc.filestoreType,
			}

			assert.Equal(t, tc.expectInternal, installation.InternalFilestore())
		})
	}
}

func TestIsSupportedFilestore(t *testing.T) {
	var testCases = []struct {
		filestore       string
		expectSupported bool
	}{
		{"", false},
		{"unknown", false},
		{model.InstallationFilestoreMinioOperator, true},
		{model.InstallationFilestoreAwsS3, true},
		{model.InstallationFilestoreMultiTenantAwsS3, true},
		{model.InstallationFilestoreBifrost, true},
		{model.InstallationFilestoreLocalEphemeral, true},
	}

	for _, tc := range testCases {
		t.Run(tc.filestore, func(t *testing.T) {
			assert.Equal(t, tc.expectSupported, model.IsSupportedFilestore(tc.filestore))
		})
	}
}

func TestLocalEphemeralFilestore(t *testing.T) {
	logger := log.New()
	logger.SetOutput(io.Discard)

	t.Run("NewLocalEphemeralFilestore", func(t *testing.T) {
		filestore := model.NewLocalEphemeralFilestore()
		assert.NotNil(t, filestore)
		assert.IsType(t, &model.LocalEphemeralFilestore{}, filestore)
	})

	t.Run("Provision", func(t *testing.T) {
		filestore := model.NewLocalEphemeralFilestore()
		err := filestore.Provision(nil, logger)
		assert.NoError(t, err)
	})

	t.Run("Teardown", func(t *testing.T) {
		filestore := model.NewLocalEphemeralFilestore()
		err := filestore.Teardown(false, nil, logger)
		assert.NoError(t, err)
	})

	t.Run("Teardown with keepData", func(t *testing.T) {
		filestore := model.NewLocalEphemeralFilestore()
		err := filestore.Teardown(true, nil, logger)
		assert.NoError(t, err)
	})

	t.Run("GenerateFilestoreSpecAndSecret", func(t *testing.T) {
		filestore := model.NewLocalEphemeralFilestore()
		config, secret, err := filestore.GenerateFilestoreSpecAndSecret(nil, logger)
		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.NotNil(t, secret)
	})

	t.Run("implements Filestore interface", func(t *testing.T) {
		var _ model.Filestore = (*model.LocalEphemeralFilestore)(nil)
		var _ model.Filestore = model.NewLocalEphemeralFilestore()
	})
}
