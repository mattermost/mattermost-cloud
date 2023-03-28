// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"os"
)

type WebhookHeader struct {
	Key          string  `json:"key"`
	Value        *string `json:"value,omitempty"`
	ValueFromEnv *string `json:"value_from_env,omitempty"`
}

type Headers []WebhookHeader

func (wh Headers) Value() (driver.Value, error) {
	return json.Marshal(wh)
}

func (wh *Headers) Scan(databaseValue interface{}) error {
	switch value := databaseValue.(type) {
	case string: // sqlite's text
		return json.Unmarshal([]byte(value), wh)
	case []byte: // psqls jsonb
		return json.Unmarshal(value, wh)
	case nil:
		return nil
	default:
		return fmt.Errorf("cannot scan type %t into Headers", databaseValue)
	}
}

func (wh Headers) Validate() error {
	keys := make(map[string]struct{}, len(wh))
	for _, header := range wh {
		if _, ok := keys[header.Key]; ok {
			return fmt.Errorf("header %s is duplicated", header.Key)
		}
		keys[header.Key] = struct{}{}
		if header.Value == nil && header.ValueFromEnv == nil {
			return fmt.Errorf("header %s must have either a value or a value_from_env", header.Key)
		}
		if header.Value != nil && header.ValueFromEnv != nil {
			return fmt.Errorf("header %s cannot have both a value and a value_from_env", header.Key)
		}
	}
	return nil
}

func (wh Headers) GetHeaders() map[string]string {
	headers := make(map[string]string, len(wh))
	for _, header := range wh {
		if header.Value != nil {
			headers[header.Key] = *header.Value
		} else if header.ValueFromEnv != nil {
			headers[header.Key] = os.Getenv(*header.ValueFromEnv)
		}
	}
	return headers
}
