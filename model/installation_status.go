// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"
)

// InstallationsStatus represents the status of all installations.
type InstallationsStatus struct {
	InstallationsTotal       int64
	InstallationsStable      int64
	InstallationsHibernating int64
	InstallationsUpdating    int64
}

// InstallationsStatusFromReader decodes a json-encoded InstallationsStatus from the given io.Reader.
func InstallationsStatusFromReader(reader io.Reader) (*InstallationsStatus, error) {
	installationsStatus := InstallationsStatus{}
	decoder := json.NewDecoder(reader)
	err := decoder.Decode(&installationsStatus)
	if err != nil && err != io.EOF {
		return nil, err
	}

	return &installationsStatus, nil
}
