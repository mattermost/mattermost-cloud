// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

//+build e2e

package pkg

import (
	"fmt"
	"math/rand"
)

const (
	installationDNSFormat = "e2e-test-%s.%s"
)

// GetDNS returns e2e test dns with random suffix.
func GetDNS(domain string) string {
	return fmt.Sprintf(installationDNSFormat, randStringBytes(4), domain)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyz"

func randStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}
