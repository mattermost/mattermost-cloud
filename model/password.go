// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"math/rand"
	"time"
)

const (
	passwordBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"

	// DefaultPasswordLength the default password length used when calling the below function,
	// mainly used on database creation.
	DefaultPasswordLength = 40
)

// NewRandomPassword generates a random password of the provided length
func NewRandomPassword(length int) string {
	rand.Seed(time.Now().UnixNano())

	b := make([]byte, length)
	for i := range b {
		b[i] = passwordBytes[rand.Intn(len(passwordBytes))]
	}

	return string(b)
}
