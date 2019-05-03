package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBasename(t *testing.T) {
	var basenameTests = []struct {
		file     ManifestFile
		expected string
	}{
		{
			ManifestFile{Path: "/tmp/test/1/file1.yaml"},
			"file1.yaml",
		}, {
			ManifestFile{Path: "noDirectory.yaml"},
			"noDirectory.yaml",
		},
	}

	for _, tt := range basenameTests {
		t.Run(tt.file.Path, func(t *testing.T) {
			assert.Equal(t, tt.file.Basename(), tt.expected)
		})
	}
}
