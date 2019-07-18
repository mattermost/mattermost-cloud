package model_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/model"
	"github.com/stretchr/testify/require"
)

func TestNewAWSMetadata(t *testing.T) {
	t.Run("nil payload", func(t *testing.T) {
		awsMetadata, err := model.NewAWSMetadata(nil)
		require.NoError(t, err)
		require.Empty(t, awsMetadata.Zones)
	})

	t.Run("invalid payload", func(t *testing.T) {
		_, err := model.NewAWSMetadata([]byte(`{`))
		require.Error(t, err)
	})

	t.Run("valid payload", func(t *testing.T) {
		awsMetadata, err := model.NewAWSMetadata([]byte(`{"Zones": ["zone1", "zone2"]}`))
		require.NoError(t, err)
		require.Equal(t, []string{"zone1", "zone2"}, awsMetadata.Zones)
	})
}
