// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor_test

import (
	"fmt"
	"testing"

	"github.com/mattermost/mattermost-cloud/internal/supervisor"
	"github.com/mattermost/mattermost-cloud/internal/testlib"
	"github.com/stretchr/testify/require"
)

type testDoer struct {
	calls chan bool
}

func (td *testDoer) Do() error {
	td.calls <- true

	return nil
}

func (td *testDoer) Shutdown() {}

type failDoer struct {
}

func (fd *failDoer) Do() error {
	return fmt.Errorf("failed")
}

func (td *failDoer) Shutdown() {}

func TestMultiDoer(t *testing.T) {
	logger := testlib.MakeLogger(t)
	t.Run("failure", func(t *testing.T) {
		d1 := &testDoer{calls: make(chan bool, 1)}
		d2 := &failDoer{}
		d3 := &testDoer{calls: make(chan bool, 1)}

		doer := supervisor.NewMultiDoer(logger)
		doer.Append(d1, d2, d3)

		err := doer.Do()
		require.EqualError(t, err, "doers failed, check previous logs for details")

		select {
		case <-d1.calls:
		default:
			require.Fail(t, "doer1 not invoked")
		}

		select {
		case <-d3.calls:
		default:
			require.Fail(t, "doer3 not invoked")
		}
	})

	t.Run("success", func(t *testing.T) {
		d1 := &testDoer{calls: make(chan bool, 1)}
		d2 := &testDoer{calls: make(chan bool, 1)}
		d3 := &testDoer{calls: make(chan bool, 1)}

		doer := supervisor.NewMultiDoer(logger)
		doer.Append(d1, d2, d3)

		err := doer.Do()
		require.NoError(t, err)

		select {
		case <-d1.calls:
		default:
			require.Fail(t, "doer1 not invoked")
		}

		select {
		case <-d2.calls:
		default:
			require.Fail(t, "doer2 not invoked")
		}

		select {
		case <-d3.calls:
		default:
			require.Fail(t, "doer3 not invoked")
		}
	})
}
