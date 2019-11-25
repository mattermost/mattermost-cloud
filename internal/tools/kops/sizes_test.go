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
		{"SizeAlefDev", sizeAlefDev, false},
		{"SizeAlef500", sizeAlef500, false},
		{"SizeAlef1000", sizeAlef1000, false},
		{"SizeAlef5000", sizeAlef5000, false},
		{"SizeAlef10000", sizeAlef10000, false},
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

	var haSizeTests = []struct {
		size        string
		clusterSize ClusterSize
		masterCount string
		expectError bool
	}{
		{"SizeAlef500-HA3", sizeAlef500, "3", false},
		{"SizeAlef1000-HA2", sizeAlef1000, "2", false},
		{"SizeAlef5000-HA3", sizeAlef5000, "3", false},
		{"SizeAlef500-HA4", ClusterSize{}, "", true},
		{"SizeAlef500-HA", ClusterSize{}, "", true},
		{"SizeAlef500-HA3-HA2", ClusterSize{}, "", true},
	}

	for _, tt := range haSizeTests {
		t.Run(tt.size, func(t *testing.T) {
			clusterSize, err := GetSize(tt.size)
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			haClusterSize := tt.clusterSize
			haClusterSize.MasterCount = tt.masterCount
			assert.Equal(t, clusterSize, haClusterSize)
		})
	}
}
