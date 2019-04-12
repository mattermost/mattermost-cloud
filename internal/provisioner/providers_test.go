package provisioner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

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
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
