package terraform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestArg(t *testing.T) {
	var sizeTests = []struct {
		key      string
		vals     []string
		expected string
	}{
		{
			"1valNoKeyDash",
			[]string{"val1"},
			"-1valNoKeyDash=val1",
		}, {
			"3valsNoKeyDash",
			[]string{"val1", "val2", "val3"},
			"-3valsNoKeyDash=val1val2val3",
		}, {
			"-3valsWithKeyDash",
			[]string{"val1", "val2", "val3"},
			"-3valsWithKeyDash=val1val2val3",
		}, {
			"KeyNoVal",
			[]string{},
			"-KeyNoVal",
		},
	}

	for _, tt := range sizeTests {
		t.Run(tt.key, func(t *testing.T) {
			assert.Equal(t, arg(tt.key, tt.vals...), tt.expected)
		})
	}
}
