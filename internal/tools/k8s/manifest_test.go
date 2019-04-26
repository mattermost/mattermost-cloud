package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFQN(t *testing.T) {
	var fqnTests = []struct {
		file     ManifestFile
		expected string
	}{
		{
			ManifestFile{
				Name:      "file1",
				Directory: "/tmp/test/1",
			},
			"/tmp/test/1/file1",
		},
	}

	for _, tt := range fqnTests {
		t.Run(tt.file.Name, func(t *testing.T) {
			assert.Equal(t, tt.file.FQN(), tt.expected)
		})
	}
}
