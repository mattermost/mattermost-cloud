package model_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/model"
	"github.com/stretchr/testify/require"
)

func TestNewAWSMetadata(t *testing.T) {
	awsMetadata := model.NewAWSMetadata(nil)
	require.Empty(t, awsMetadata.Zones)

	awsMetadata = model.NewAWSMetadata([]byte(`{"Zones": ["zone1", "zone2"]}`))
	require.Equal(t, []string{"zone1", "zone2"}, awsMetadata.Zones)
}
