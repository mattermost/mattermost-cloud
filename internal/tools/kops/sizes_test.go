package kops

import "testing"

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
			if err != nil && !tt.expectError {
				t.Errorf("got error when expecting none: %s", err)
			}
			if err == nil && tt.expectError {
				t.Errorf("expecting error, got none")
			}
			if clusterSize != tt.clusterSize {
				t.Errorf("got %+v, want %+v", clusterSize, tt.clusterSize)
			}
		})
	}
}
