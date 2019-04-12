package kops

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSize(t *testing.T) {
	var sizeTests = []struct {
		size        string
		clusterSize ClusterSize
		expectError bool
	}{
		{"SizeAlef500", sizeAlef500, false},
		{"SizeAlef1000", sizeAlef1000, false},
		{"IncorrectSize", ClusterSize{}, true},
	}

	for _, tt := range sizeTests {
		t.Run(tt.size, func(t *testing.T) {
			clusterSize, err := GetSize(tt.size)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, clusterSize, tt.clusterSize)
		})
	}
}
