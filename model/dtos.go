// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
)

func DTOFromReader[T any](reader io.Reader) (*T, error) {
	var dto T
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&dto)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &dto, nil
}

func DTOsFromReader[T any](reader io.Reader) ([]*T, error) {
	dtos := []*T{}
	decoder := json.NewDecoder(reader)

	err := decoder.Decode(&dtos)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return dtos, nil
}
