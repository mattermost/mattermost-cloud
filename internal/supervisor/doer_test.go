package supervisor_test

import (
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/stretchr/testify/require"
)

type testDoer struct {
	calls chan bool
}

func (td *testDoer) Do() error {
	td.calls <- true

	return nil
}

type failDoer struct {
}

func (fd *failDoer) Do() error {
	return fmt.Errorf("failed")
}

func TestMultiDoer(t *testing.T) {
	d1 := &testDoer{calls: make(chan bool, 1)}
	d2 := &failDoer{}
	d3 := &testDoer{calls: make(chan bool)}

	doer := supervisor.MultiDoer{d1, d2, d3}

	err := doer.Do()
	require.EqualError(t, err, "failed")

	select {
	case <-d1.calls:
	default:
		require.Fail(t, "doer1 not invoked")
	}

	select {
	case <-d3.calls:
		require.Fail(t, "doer3 should not be invoked")
	default:
	}
}
