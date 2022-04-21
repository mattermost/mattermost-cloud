// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package cloudflare

import (
	"context"
	"strings"
	"time"

	cf "github.com/cloudflare/cloudflare-go"
	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"
)

const defaultTimeout = 30 * time.Second

func (c *Client) getZoneID(zoneName string) (zoneID string, err error) {
	zoneID, err = c.cfClient.ZoneIDByName(zoneName)
	if err != nil {
		return "", err
	}

	return zoneID, nil
}

func (c *Client) getZoneName(zoneNameList []string, customerDNSName string) (zoneName string, found bool) {
	for _, zoneName := range zoneNameList {
		if zoneName == "" {
			return "", false
		}
		if strings.HasSuffix(customerDNSName, zoneName) {
			return zoneName, true
		}
	}
	return "", false
}

func (c *Client) getRecordID(zoneID, customerDNSName string, logger logrus.FieldLogger) (recordID string, err error) {

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	dnsRecords, err := c.cfClient.DNSRecords(ctx, zoneID, cf.DNSRecord{Name: customerDNSName})
	if err != nil {
		return "", errors.Wrap(err, "failed to get DNS Record ID from Cloudflare")
	}
	if len(dnsRecords) == 0 {
		logger.Info("Unable to find any DNS records in Cloudflare; skipping...")
		return "", nil
	}

	return dnsRecords[0].ID, nil

}

// CreateDNSRecord creates a DNS record in the first given Cloudflare zone name of the list
func (c *Client) CreateDNSRecord(customerDNSName string, dnsEndpoints []string, logger logrus.FieldLogger) error {
	zoneNameList := c.aws.GetPublicHostedZoneNames()
	if len(zoneNameList) == 0 {
		return errors.New("no public hosted zones names found from AWS")
	}

	if len(dnsEndpoints) == 0 {
		return errors.New("no DNS endpoints provided for Cloudflare creation request")
	}
	dnsEndpoint := dnsEndpoints[0]
	if dnsEndpoint == "" {
		return errors.New("DNS endpoint was an empty string")
	}

	// Fetch the zone name for that customer DNS name
	zoneName, found := c.getZoneName(zoneNameList, customerDNSName)
	if !found {
		return errors.Errorf("hosted zone for %q domain name not found", customerDNSName)
	}

	// Fetch the zone ID
	zoneID, err := c.getZoneID(zoneName)
	if err != nil {
		return errors.Wrap(err, "failed to fetch Zone ID from Cloudflare")
	}

	proxied := true

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	recordResp, err := c.cfClient.CreateDNSRecord(ctx, zoneID, cf.DNSRecord{
		Name:    customerDNSName,
		Type:    "CNAME",
		Content: dnsEndpoint,
		TTL:     1,
		Proxied: &proxied,
	})
	if err != nil {
		return errors.Wrap(err, "failed to create DNS Record at Cloudflare")
	}

	logger.WithFields(logrus.Fields{
		"cloudflare-dns-value":    customerDNSName,
		"cloudflare-dns-endpoint": dnsEndpoint,
		"cloudflare-zone-id":      zoneID,
	}).Debugf("Cloudflare create DNS record response: %v", recordResp)

	return nil
}

// DeleteDNSRecord gets DNS name and zone name which uses to delete that DNS record from Cloudflare
func (c *Client) DeleteDNSRecord(customerDNSName string, logger logrus.FieldLogger) error {
	zoneNameList := c.aws.GetPublicHostedZoneNames()
	if len(zoneNameList) == 0 {
		return errors.New("no public hosted zones names found from AWS")
	}
	// Fetch the zone name for that customer DNS name
	zoneName, found := c.getZoneName(zoneNameList, customerDNSName)
	if !found {
		return errors.Errorf("hosted zone for %q domain name not found", customerDNSName)
	}

	// Fetch the zone ID
	zoneID, err := c.getZoneID(zoneName)
	if err != nil {
		return errors.Wrap(err, "failed to fetch Zone ID from Cloudflare")
	}

	recordID, err := c.getRecordID(zoneID, customerDNSName, logger)
	if err != nil {
		return errors.Wrapf(err, "Failed to get record ID from Cloudflare for DNS: %s", customerDNSName)
	}

	// Unable to find any record, skipping deletion
	if recordID == "" {
		logger.Info("Unable to find any DNS records in Cloudflare; skipping...")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	err = c.cfClient.DeleteDNSRecord(ctx, zoneID, recordID)
	if err != nil {
		return errors.Wrap(err, "Failed to delete DNS Record at Cloudflare")
	}
	return nil
}
