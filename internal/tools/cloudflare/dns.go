// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package cloudflare

import (
	"context"
	"fmt"
	"strings"
	"time"

	cf "github.com/cloudflare/cloudflare-go"
	"github.com/pkg/errors"

	"github.com/sirupsen/logrus"
)

const defaultTimeout = 30 * time.Second

type Cloudflarer interface {
	ZoneIDByName(zoneName string) (string, error)
	DNSRecords(ctx context.Context, zoneID string, rr cf.DNSRecord) ([]cf.DNSRecord, error)
	CreateDNSRecord(ctx context.Context, zoneID string, rr cf.DNSRecord) (*cf.DNSRecordResponse, error)
	DeleteDNSRecord(ctx context.Context, zoneID, recordID string) error
}

type Client struct {
	cfClient Cloudflarer
}

// NewClientWithToken creates a new client that can be used to run the other functions.
func NewClientWithToken(client Cloudflarer) *Client {
	return &Client{
		cfClient: client,
	}
}

func (c *Client) getZoneId(zoneName string) (zoneID string, err error) {
	zoneID, err = c.cfClient.ZoneIDByName(zoneName)
	if err != nil {
		return "", err
	}

	return zoneID, err
}

func (c *Client) getZoneName(zoneNameList []string, customerDnsName string) (zoneName string, found bool) {
	for _, zoneName := range zoneNameList {
		if strings.HasSuffix(customerDnsName, zoneName) {
			return zoneName, true
		}
	}
	return "", false
}

func (c *Client) getRecordId(zoneID, customerDnsName string, logger logrus.FieldLogger) (recordID string, err error) {

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	dnsRecords, err := c.cfClient.DNSRecords(ctx, zoneID, cf.DNSRecord{Name: customerDnsName})
	if err != nil {
		logger.WithError(err).Error("failed to get DNS Record ID from Cloudflare")
		return "", err
	}
	if len(dnsRecords) == 0 {
		logger.Info("Unable to find any DNS records in Cloudflare; skipping...")
		return "", nil
	}

	return dnsRecords[0].ID, nil

}

func (c *Client) CreateDNSRecord(customerDnsName string, zoneNameList []string, dnsEndpoints []string, logger logrus.FieldLogger) error {

	if len(dnsEndpoints) == 0 {
		return errors.New("no DNS endpoints provided for Cloudflare creation request")
	}
	dnsEndpoint := dnsEndpoints[0]
	if dnsEndpoint == "" {
		return errors.New("DNS endpoint was an empty string")
	}

	// Fetch the zone name for that customer DNS name
	zoneName, found := c.getZoneName(zoneNameList, customerDnsName)
	if !found {
		return errors.Errorf("hosted zone for %q domain name not found", customerDnsName)
	}

	// Fetch the zone ID
	zoneID, err := c.getZoneId(zoneName)
	if err != nil {
		logger.WithError(err).Error("failed to fetch Zone ID from Cloudflare")
		return err
	}

	proxied := true

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()
	recordResp, err := c.cfClient.CreateDNSRecord(ctx, zoneID, cf.DNSRecord{
		Name:    customerDnsName,
		Type:    "CNAME",
		Content: dnsEndpoint,
		TTL:     1,
		Proxied: &proxied,
	})
	if err != nil {
		logger.WithError(err).Error("failed to create DNS Record at Cloudflare")
		return err
	}
	fmt.Println(recordResp)

	logger.WithFields(logrus.Fields{
		"cloudflare-dns-value":    customerDnsName,
		"cloudflare-dns-endpoint": dnsEndpoint,
		"cloudflare-zone-id":      zoneID,
	}).Debugf("Cloudflare create DNS record response: %v", recordResp)

	return nil
}

// DeleteDNSRecord gets DNS name and zone name which uses to delete that DNS record from Cloudflare
func (c *Client) DeleteDNSRecord(customerDnsName string, zoneNameList []string, logger logrus.FieldLogger) error {

	// Fetch the zone name for that customer DNS name
	zoneName, found := c.getZoneName(zoneNameList, customerDnsName)
	if !found {
		return errors.Errorf("hosted zone for %q domain name not found", customerDnsName)
	}

	// Fetch the zone ID
	zoneID, err := c.getZoneId(zoneName)
	if err != nil {
		logger.WithError(err).Error("failed to fetch Zone ID from Cloudflare")
		return err
	}

	recordID, err := c.getRecordId(zoneID, customerDnsName, logger)
	if err != nil {
		logger.WithError(err).Errorf("Failed to get record ID from Cloudflare for DNS: %s", customerDnsName)
		return err
	}

	// Unable to find any record, skipping deletion
	if err == nil && recordID == "" {
		logger.Info("Unable to find any DNS records in Cloudflare; skipping...")
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	err = c.cfClient.DeleteDNSRecord(ctx, zoneID, recordID)
	if err != nil {
		logger.WithError(err).Error("Failed to delete DNS Record at Cloudflare")
		return err
	}
	return nil
}
