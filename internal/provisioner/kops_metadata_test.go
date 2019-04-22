package provisioner

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewKopsMetadata(t *testing.T) {
	kopsMetadata := NewKopsMetadata(nil)
	require.Equal(t, "", kopsMetadata.Name)

	kopsMetadata = NewKopsMetadata([]byte(`{"Name": "name"}`))
	require.Equal(t, "name", kopsMetadata.Name)
}
