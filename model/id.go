// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"bytes"
	"encoding/base32"
	"math/rand"
	"time"
	"unicode"

	"github.com/pborman/uuid"
)

var encoding = base32.NewEncoding("ybndrfg8ejkmcpqxot1uwisza345h769")

// NewID is a globally unique identifier.  It is a [A-Z0-9] string 26
// characters long.  It is a UUID version 4 Guid that is zbased32 encoded
// with the padding stripped off.
func NewID() string {
	var b bytes.Buffer
	encoder := base32.NewEncoder(encoding, &b)
	encoder.Write(uuid.NewRandom())
	encoder.Close()
	b.Truncate(26) // removes the '==' padding
	return b.String()
}

// ClusterNewID is a globally unique identifier for cluster ID which start with a letter.  It is a [a-z0-9] string 26
func ClusterNewID() string {
	strID := NewID()
	if unicode.IsNumber(rune(strID[0])) {
		//Generate a lower case random character between a to z
		rand.Seed(time.Now().Unix())
		randomChar := 'a' + rune(rand.Intn(26))
		strID = string(randomChar) + strID[1:]
	}
	return strID
}
