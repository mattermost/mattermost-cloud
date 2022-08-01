// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package supervisor

import (
	"github.com/mattermost/mattermost-cloud/internal/tools/aws"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// Route53DNSProvider wraps route53 function calls to implement
// InstallationDNSProvider interface.
type Route53DNSProvider struct {
	aws aws.AWS
}

// NewRoute53DNSProvider creates Route53 Installation DNS Provider.
func NewRoute53DNSProvider(aws aws.AWS) *Route53DNSProvider {
	return &Route53DNSProvider{aws: aws}
}

// CreateDNSRecords creates or updates Route53 CNAME records.
func (r *Route53DNSProvider) CreateDNSRecords(customerDNSName []string, dnsEndpoints []string, logger logrus.FieldLogger) error {
	return r.aws.UpsertPublicCNAMEs(customerDNSName, dnsEndpoints, logger)
}

// DeleteDNSRecords deletes Route53 CNAME records.
func (r *Route53DNSProvider) DeleteDNSRecords(customerDNSName []string, logger logrus.FieldLogger) error {
	return r.aws.DeletePublicCNAMEs(customerDNSName, logger)
}

// DNSManager wraps multiple InstallationDNSProviders.
type DNSManager struct {
	providers []InstallationDNSProvider
}

// NewDNSManager creates new DNSManager without any providers.
func NewDNSManager() *DNSManager {
	return &DNSManager{providers: []InstallationDNSProvider{}}
}

// AddProvider adds InstallationDNSProvider to the DNSManager.
func (dm *DNSManager) AddProvider(provider InstallationDNSProvider) {
	dm.providers = append(dm.providers, provider)
}

// IsValid verifies if DNS providers are registered with DNSManager.
func (dm *DNSManager) IsValid() error {
	if len(dm.providers) == 0 {
		return errors.Errorf("error: no Installation DNS providers registerd")
	}
	return nil
}

// CreateDNSRecords creates DNS records with all registered providers.
func (dm *DNSManager) CreateDNSRecords(customerDNSName []string, dnsEndpoints []string, logger logrus.FieldLogger) error {
	for _, provider := range dm.providers {
		err := provider.CreateDNSRecords(customerDNSName, dnsEndpoints, logger)
		if err != nil {
			return errors.Wrap(err, "failed to create DNS record for one of the providers")
		}
	}
	return nil
}

// DeleteDNSRecords deletes DNS record from all registered providers.
func (dm *DNSManager) DeleteDNSRecords(customerDNSName []string, logger logrus.FieldLogger) error {
	for _, provider := range dm.providers {
		err := provider.DeleteDNSRecords(customerDNSName, logger)
		if err != nil {
			return errors.Wrap(err, "failed to delete DNS record for one of the providers")
		}
	}
	return nil
}
