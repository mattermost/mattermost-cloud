package util_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSToP(t *testing.T) {
	t.Run("Convert string to *string", func(t *testing.T) {
		input := "test"
		result := util.SToP(input)
		require.NotNil(t, result, "Expected non-nil result for SToP")
		assert.Equal(t, input, *result, "Expected dereferenced value to match input")
	})
}

func TestIToP(t *testing.T) {
	t.Run("Convert int64 to *int64", func(t *testing.T) {
		var input int64 = 123
		result := util.IToP(input)
		require.NotNil(t, result, "Expected non-nil result for IToP")
		assert.Equal(t, input, *result, "Expected dereferenced value to match input")
	})
}

func TestBToP(t *testing.T) {
	t.Run("Convert bool to *bool", func(t *testing.T) {
		input := true
		result := util.BToP(input)
		require.NotNil(t, result, "Expected non-nil result for BToP")
		assert.Equal(t, input, *result, "Expected dereferenced value to match input")
	})
}
