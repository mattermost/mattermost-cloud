// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// StringMap type used to have a map[string]string directly in the database in a TEXT or JSON/JSONB
// field
type StringMap map[string]string

func (m StringMap) Value() (driver.Value, error) {
	return json.Marshal(m)
}

func (m StringMap) Scan(v interface{}) error {
	if v == nil {
		return nil
	}
	switch data := v.(type) {
	case string: // sqlite's text
		return json.Unmarshal([]byte(data), &m)
	case []byte: // psqls jsonb
		return json.Unmarshal(data, &m)
	default:
		return fmt.Errorf("cannot scan type %t into Map", v)
	}
}
