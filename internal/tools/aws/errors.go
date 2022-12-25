// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"github.com/aws/smithy-go"
	"github.com/pkg/errors"
)

// IsErrorCode asserts that an AWS error has a certain code.
func IsErrorCode(err error, code string) bool {
	var awsErr smithy.APIError
	if err != nil && errors.As(err, &awsErr) {
		return awsErr.ErrorCode() == code
	}
	return false
}

// IsErrorResourceNotFound asserts that an AWS error is
// ResourceNotFoundException.
func IsErrorResourceNotFound(err error) bool {
	return IsErrorCode(err, "ResourceNotFoundException")
}

// IsErrorPermissionNotFound asserts that an AWS error is
// InvalidPermission.NotFound.
func IsErrorPermissionNotFound(err error) bool {
	return IsErrorCode(err, "InvalidPermission.NotFound")
}

// IsErrorPermissionDuplicate asserts that an AWS error is
// InvalidPermission.Duplicate.
func IsErrorPermissionDuplicate(err error) bool {
	return IsErrorCode(err, "InvalidPermission.Duplicate")
}

// IsErrorResourceInUseException asserts that an AWS error is
// ResourceInUseException.
func IsErrorResourceInUseException(err error) bool {
	return IsErrorCode(err, "ResourceInUseException")
}
