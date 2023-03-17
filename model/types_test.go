// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringMap(t *testing.T) {
	t.Run("value", func(t *testing.T) {
		testCases := []struct {
			input *StringMap
			value any
			err   bool
		}{
			{
				input: &StringMap{
					"foo": "bar",
				},
				value: []byte(`{"foo":"bar"}`),
				err:   false,
			},
			{
				input: &StringMap{},
				value: []byte(`{}`),
				err:   false,
			},
		}

		for _, test := range testCases {
			value, err := test.input.Value()
			require.Equal(t, test.value, value)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		}
	})

	t.Run("scan", func(t *testing.T) {
		testCases := []struct {
			input any
			scan  StringMap
			err   bool
		}{
			{
				input: []byte(`{"foo":"bar"}`),
				scan: StringMap{
					"foo": "bar",
				},
				err: false,
			},
			{
				input: `{"foo":"bar"}`,
				scan: StringMap{
					"foo": "bar",
				},
				err: false,
			},
		}

		for _, test := range testCases {
			stringMap := StringMap{}
			err := stringMap.Scan(test.input)
			require.Equal(t, test.scan, stringMap)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		}
	})
}
