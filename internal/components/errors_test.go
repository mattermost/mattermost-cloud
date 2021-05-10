// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package components

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestErrWithStatus(t *testing.T) {
	t.Run("wrap in regular error and downcast", func(t *testing.T) {
		err := errors.New("test")
		err = ErrWrap(400, err, "test error 2")

		err = errors.Wrap(err, "test error 3")
		err = errors.Wrap(err, "test error 4")
		err = errors.Wrap(err, "test error 5")

		status := ErrToStatus(err)
		assert.Equal(t, 400, status)
		assert.Equal(t, "test error 5: test error 4: test error 3: test error 2: test", err.Error())
	})

	t.Run("wrap error with status and downcast", func(t *testing.T) {
		err := NewErr(404, errors.New("error"))

		err = errors.Wrap(err, "test error 2")
		err = errors.Wrap(err, "test error 3")

		status := ErrToStatus(err)
		assert.Equal(t, 404, status)
		assert.Equal(t, "test error 3: test error 2: error", err.Error())
	})

	t.Run("wrap with error with status and get latest status", func(t *testing.T) {
		err := NewErr(404, errors.New("error"))

		err = ErrWrapf(400, err, "old error status: %d", ErrToStatus(err))

		status := ErrToStatus(err)
		assert.Equal(t, 400, status)
		assert.Equal(t, "old error status: 404: error", err.Error())
	})
}
