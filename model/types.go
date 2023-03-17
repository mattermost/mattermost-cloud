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
// field.
type StringMap map[string]string

func (sm StringMap) Value() (driver.Value, error) {
	return json.Marshal(sm)
}

func (sm *StringMap) Scan(databaseValue interface{}) error {
	switch value := databaseValue.(type) {
	case string: // sqlite's text
		return json.Unmarshal([]byte(value), sm)
	case []byte: // psqls jsonb
		return json.Unmarshal(value, sm)
	default:
		return fmt.Errorf("cannot scan type %t into StringMap", databaseValue)
	}
}
