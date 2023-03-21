// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

// InstallationDTO represents a Mattermost installation. DTO stands for Data Transfer Object.
type InstallationDTO struct {
	*Installation
	Annotations []*Annotation `json:"Annotations,omitempty"`
	// Deprecated: This is for backward compatibility until we switch all clients, as DNS was removed from Installation.
	DNS        string
	DNSRecords []*InstallationDNS
}
