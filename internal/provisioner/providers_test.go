package provisioner

import "testing"

func TestCheckProvider(t *testing.T) {
	var sizeTests = []struct {
		provider    string
		expectError bool
	}{
		{"aws", false},
		{"gce", true},
		{"azure", true},
	}

	for _, tt := range sizeTests {
		t.Run(tt.provider, func(t *testing.T) {
			err := checkProvider(tt.provider)
			if err != nil && !tt.expectError {
				t.Errorf("got error when expecting none: %s", err)
			}
			if err == nil && tt.expectError {
				t.Errorf("expecting error, got none")
			}
		})
	}
}
