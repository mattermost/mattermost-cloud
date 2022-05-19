// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package model

import (
	"encoding/json"
	"io"

	"github.com/pkg/errors"
)

// InstallationDNS represents single domain name for given Installation.
type InstallationDNS struct {
	ID             string
	DomainName     string
	InstallationID string
	IsPrimary      bool
	CreateAt       int64
	DeleteAt       int64
}

// AddDNSRecordRequest represents request body for adding domain name to Installation.
type AddDNSRecordRequest struct {
	DNS string
}

// Validate validates AddDNSRecordRequest.
func (request *AddDNSRecordRequest) Validate(installationName string) error {
	err := isValidDNS(request.DNS)
	if err != nil {
		return errors.Wrap(err, "dns is invalid")
	}

	return ensureDNSMatchesName(request.DNS, installationName)
}

// NewAddDNSRecordRequestFromReader will create a AddDNSRecordRequest from an io.Reader with JSON data.
func NewAddDNSRecordRequestFromReader(reader io.Reader) (*AddDNSRecordRequest, error) {
	var addDNSRecordRequest AddDNSRecordRequest
	err := json.NewDecoder(reader).Decode(&addDNSRecordRequest)
	if err != nil && err != io.EOF {
		return nil, errors.Wrap(err, "failed to decode add installation DNS record request")
	}

	return &addDNSRecordRequest, nil
}

// DNSNamesFromRecords extract slice of domain names from slice of DNSRecords.
func DNSNamesFromRecords(dnsRecords []*InstallationDNS) []string {
	strs := make([]string, 0, len(dnsRecords))
	for _, r := range dnsRecords {
		strs = append(strs, r.DomainName)
	}
	return strs
}
