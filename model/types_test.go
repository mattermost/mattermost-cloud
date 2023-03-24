// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHeaders(t *testing.T) {
	headerValue := "bar"

	t.Run("value", func(t *testing.T) {
		testCases := []struct {
			input *Headers
			value any
			err   bool
		}{
			{
				input: &Headers{{
					Key: "foo", Value: &headerValue,
				}},
				value: []byte(`[{"key":"foo","value":"bar"}]`),
				err:   false,
			},
			{
				input: &Headers{},
				value: []byte(`[]`),
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
			scan  Headers
			err   bool
		}{
			{
				input: []byte(`[{"key":"foo", "value": "bar"}]`),
				scan: Headers{{
					Key: "foo", Value: &headerValue,
				}},
				err: false,
			},
			{
				input: `[{"key": "foo", "value": "bar"}]`,
				scan: Headers{{
					Key: "foo", Value: &headerValue,
				}},
				err: false,
			},
			{
				input: nil,
				scan:  Headers{},
				err:   false,
			},
		}

		for _, test := range testCases {
			headers := Headers{}
			err := headers.Scan(test.input)
			require.Equal(t, test.scan, headers)
			if test.err {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		}
	})
}
