package model_test

import (
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/model"
	"github.com/stretchr/testify/require"
)

func TestNewKopsMetadata(t *testing.T) {
	kopsMetadata := model.NewKopsMetadata(nil)
	require.Equal(t, "", kopsMetadata.Name)

	kopsMetadata = model.NewKopsMetadata([]byte(`{"Name": "name"}`))
	require.Equal(t, "name", kopsMetadata.Name)
}
