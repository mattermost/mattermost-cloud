// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package main

import (
	"bytes"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate(t *testing.T) {
	t.Run("all generators", func(t *testing.T) {
		expectedCode, err := os.ReadFile("testdata/all_generators.txt")
		require.NoError(t, err)

		mainCmd := newRootCmd()

		outBuff := &bytes.Buffer{}
		mainCmd.SetArgs([]string{"generate", "--boilerplate-file=../../hack/boilerplate/boilerplate.generatego.txt", "--type=github.com/mattermost/mattermost-cloud/cmd/provisioner-code-gen/testdata.TestStruct", "--generator=get_id,get_state,is_deleted,as_resources"})
		mainCmd.SetOut(outBuff)

		err = mainCmd.Execute()
		require.NoError(t, err)
		assert.Equal(t, string(expectedCode), outBuff.String())
	})

	t.Run("id generator only", func(t *testing.T) {
		expectedCode, err := os.ReadFile("testdata/id_generator.txt")
		require.NoError(t, err)

		mainCmd := newRootCmd()

		outBuff := &bytes.Buffer{}
		mainCmd.SetArgs([]string{"generate", "--boilerplate-file=../../hack/boilerplate/boilerplate.generatego.txt", "--type=github.com/mattermost/mattermost-cloud/cmd/provisioner-code-gen/testdata.TestStruct", "--generator=get_id"})
		mainCmd.SetOut(outBuff)

		err = mainCmd.Execute()
		require.NoError(t, err)
		assert.Equal(t, string(expectedCode), outBuff.String())
	})
}
