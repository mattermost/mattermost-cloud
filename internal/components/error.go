// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package components

import (
	"net/http"

	"github.com/pkg/errors"
)

// ErrWithStatus represents error with status code.
// It can be hidden behind standard error interface and wrapped with errors.Wrap.
type ErrWithStatus struct {
	err    error
	status int
}

// Error returns error string.
func (e *ErrWithStatus) Error() string {
	return e.err.Error()
}

// NewErr creates ErrWithStatus as error interface.
func NewErr(status int, err error) error {
	return &ErrWithStatus{
		err:    err,
		status: status,
	}
}

// ErrWrap wraps an error inside ErrWithStatus with additional message.
func ErrWrap(status int, err error, message string) error {
	if err == nil {
		return nil
	}
	return &ErrWithStatus{
		err:    errors.Wrap(err, message),
		status: status,
	}
}

// ErrWrapf wraps an error inside ErrWithStatus with additional formatted message.
func ErrWrapf(status int, err error, format string, args ...interface{}) error {
	if err == nil {
		return nil
	}
	return &ErrWithStatus{
		err:    errors.Wrapf(err, format, args...),
		status: status,
	}
}

// ErrToStatus tries to extract status code from error. If the error is not ErrWithStatus returns status 500.
func ErrToStatus(err error) int {
	statusErr := &ErrWithStatus{}
	if errors.As(err, &statusErr) {
		return statusErr.status
	}
	return http.StatusInternalServerError
}
