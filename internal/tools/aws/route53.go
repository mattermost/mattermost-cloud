// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	defaultTTL             = 60
	defaultWeight          = 1
	hostedZoneIDLength     = 13
	hostedZoneResourceType = "hostedzone"
	hostedZonePrefix       = "/hostedzone/"
)

type awsHostedZone struct {
	ID string
}

type route53Cache struct {
	privateHostedZoneID string
	publicHostedZones   map[string]awsHostedZone
}

func (a *Client) buildRoute53Cache() error {
	privateID, err := a.getHostedZoneIDWithTag(Tag{
		Key:   DefaultCloudDNSTagKey,
		Value: DefaultPrivateCloudDNSTagValue,
	})
	if err != nil {
		return errors.Wrap(err, "failed to get private hosted zone ID")
	}

	publicTag := Tag{
		Key:   DefaultCloudDNSTagKey,
		Value: DefaultPublicCloudDNSTagValue,
	}

	zones, err := a.GetHostedZonesWithTag(publicTag)
	if err != nil {
		return errors.Wrap(err, "failed to get public hosted zone ID")
	}
	if len(zones) == 0 {
		return errors.Errorf("no hosted zone associated with tag: %s", publicTag.String())
	}

	zoneMap := map[string]awsHostedZone{}

	for _, zone := range zones {
		zoneID, err := parseHostedZoneResourceID(zone)
		if err != nil {
			return errors.Wrap(err, "failed to parse hosted zone ID")
		}

		zoneMap[strings.TrimSuffix(getString(zone.Name), ".")] = awsHostedZone{ID: zoneID}
	}

	a.cache.route53 = &route53Cache{
		privateHostedZoneID: privateID,
		publicHostedZones:   zoneMap,
	}

	return nil
}

func (a *Client) getDNSZoneID(dns string) (string, bool) {
	for domainName, zone := range a.cache.route53.publicHostedZones {
		if strings.HasSuffix(dns, domainName) {
			return zone.ID, true
		}
	}

	return "", false
}

// CreatePublicCNAME creates a record in Route53 for a public domain name.
func (a *Client) CreatePublicCNAME(dnsName string, dnsEndpoints []string, dnsIdentifier string, logger log.FieldLogger) error {
	zoneID, found := a.getDNSZoneID(dnsName)
	if !found {
		return errors.Errorf("hosted zone for %q domain name not found", dnsName)
	}
	return a.createCNAME(zoneID, dnsName, dnsEndpoints, dnsIdentifier, logger)
}

// UpdatePublicRecordIDForCNAME updates the record ID for the record corresponding
// to a DNS value in the public hosted zone.
func (a *Client) UpdatePublicRecordIDForCNAME(dnsName, newID string, logger log.FieldLogger) error {
	zoneID, found := a.getDNSZoneID(dnsName)
	if !found {
		return errors.Errorf("hosted zone for %q domain name not found", dnsName)
	}
	return a.updateResourceRecordIDs(zoneID, dnsName, newID, logger)
}

// DeletePublicCNAME deletes a AWS route53 record for a public domain name.
func (a *Client) DeletePublicCNAME(dnsName string, logger log.FieldLogger) error {
	zoneID, found := a.getDNSZoneID(dnsName)
	if !found {
		return errors.Errorf("hosted zone for %q domain name not found", dnsName)
	}
	return a.deleteCNAME(zoneID, dnsName, logger)
}

// GetPrivateHostedZoneID returns the private R53 hosted zone ID for the AWS
// account.
func (a *Client) GetPrivateHostedZoneID() string {
	return a.cache.route53.privateHostedZoneID
}

// GetPublicHostedZoneNames returns the public R53 hosted zone Name list for the AWS account.
func (a *Client) GetPublicHostedZoneNames() []string {
	var domainNameList []string
	for domainName, _ := range a.cache.route53.publicHostedZones {
		domainNameList = append(domainNameList, domainName)
	}
	return domainNameList
}

// CreatePrivateCNAME creates a record in Route53 for a private domain name.
func (a *Client) CreatePrivateCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error {
	return a.createCNAME(a.GetPrivateHostedZoneID(), dnsName, dnsEndpoints, "", logger)
}

// IsProvisionedPrivateCNAME returns true if a record has been registered in the
// private hosted zone for the given CNAME (full FQDN required as input)
func (a *Client) IsProvisionedPrivateCNAME(dnsName string, logger log.FieldLogger) bool {
	return a.isProvisionedCNAME(a.GetPrivateHostedZoneID(), dnsName, logger)
}

// GetPrivateZoneDomainName gets the private Route53 domain name.
func (a *Client) GetPrivateZoneDomainName(logger log.FieldLogger) (string, error) {
	return a.getZoneDNS(a.GetPrivateHostedZoneID(), logger)
}

// DeletePrivateCNAME deletes an AWS route53 record for a private domain name.
func (a *Client) DeletePrivateCNAME(dnsName string, logger log.FieldLogger) error {
	return a.deleteCNAME(a.GetPrivateHostedZoneID(), dnsName, logger)
}

// GetTagByKeyAndZoneID returns a Tag of a given tag:key and of a given route53 id
func (a *Client) GetTagByKeyAndZoneID(key string, id string, logger log.FieldLogger) (*Tag, error) {
	tagList, err := a.Service().route53.ListTagsForResource(&route53.ListTagsForResourceInput{
		ResourceId:   aws.String(id),
		ResourceType: aws.String(hostedZoneResourceType),
	})
	if err != nil {
		return nil, errors.Wrap(err, "unable to get tag list")
	}

	for _, resourceTag := range tagList.ResourceTagSet.Tags {
		if resourceTag != nil {
			resourceTagKey := resourceTag.Key
			if resourceTagKey != nil && *resourceTagKey == trimTagPrefix(key) {
				logger.WithFields(log.Fields{
					"route53-tag-key":        *resourceTag.Key,
					"route53-hosted-zone-id": id,
				}).Debug("AWS Route53 Hosted Zone Tag found")
				return &Tag{
					Key:   *resourceTag.Key,
					Value: *resourceTag.Value,
				}, nil
			}
		}
	}
	return nil, nil
}

func (a *Client) getZoneDNS(hostedZoneID string, logger log.FieldLogger) (string, error) {
	out, err := a.Service().route53.GetHostedZone(&route53.GetHostedZoneInput{
		Id: aws.String(hostedZoneID),
	})
	if err != nil {
		return "", err
	}

	domainName := trimTrailingDomainDot(*out.HostedZone.Name)

	if domainName == "" {
		return "", errors.New("the returned domain name was empty")
	}

	logger.WithFields(log.Fields{
		"route53-domain-name":    domainName,
		"route53-hosted-zone-id": hostedZoneID,
	}).Debug("AWS Route53 domain lookup complete")

	return domainName, nil
}

func (a *Client) createCNAME(hostedZoneID, dnsName string, dnsEndpoints []string, dnsIdentifier string, logger log.FieldLogger) error {
	if len(dnsEndpoints) == 0 {
		return errors.New("no DNS endpoints provided for route53 creation request")
	}
	for _, endpoint := range dnsEndpoints {
		if endpoint == "" {
			return errors.New("at least one of the DNS endpoints was set to an empty string")
		}
	}

	var resourceRecords []*route53.ResourceRecord
	for _, endpoint := range dnsEndpoints {
		resourceRecords = append(resourceRecords, &route53.ResourceRecord{
			Value: aws.String(endpoint),
		})
	}

	identifier := dnsName
	if len(dnsIdentifier) != 0 {
		identifier = dnsIdentifier
	}

	resp, err := a.Service().route53.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name:            aws.String(dnsName),
						Type:            aws.String("CNAME"),
						ResourceRecords: resourceRecords,
						TTL:             aws.Int64(defaultTTL),
						Weight:          aws.Int64(defaultWeight),
						SetIdentifier:   aws.String(identifier),
					},
				},
			},
		},
		HostedZoneId: &hostedZoneID,
	})
	if err != nil {
		return err
	}

	logger.WithFields(log.Fields{
		"route53-dns-value":      dnsName,
		"route53-dns-endpoints":  dnsEndpoints,
		"route53-hosted-zone-id": hostedZoneID,
	}).Debugf("AWS Route53 create response: %s", prettyRoute53Response(resp))

	return nil
}

func (a *Client) isProvisionedCNAME(hostedZoneID, dnsName string, logger log.FieldLogger) bool {
	recordSets, err := a.getRecordSetsForDNS(hostedZoneID, dnsName, logger)
	if err != nil {
		logger.WithError(err).Errorf("failed to get record sets for dns name %s", dnsName)
		return false
	}

	for _, recordSet := range recordSets {
		if recordSet.Name != nil && dnsName == strings.TrimRight(*recordSet.Name, ".") {
			return true
		}
	}

	return false
}

func (a *Client) updateResourceRecordIDs(hostedZoneID, dnsName, newID string, logger log.FieldLogger) error {
	recordSets, err := a.getRecordSetsForDNS(hostedZoneID, dnsName, logger)
	if err != nil {
		return errors.Wrapf(err, "failed to get record sets for dns name %s", dnsName)
	}

	// There should only be one returned record.
	if len(recordSets) != 1 {
		return errors.Errorf("expected exactly 1 resource record, but found %d", len(recordSets))
	}

	recordSet := recordSets[0]
	if *recordSet.SetIdentifier == newID {
		return nil
	}

	newRecordSet := *recordSet
	newRecordSet.SetIdentifier = aws.String(newID)

	resp, err := a.Service().route53.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action:            aws.String("UPSERT"),
					ResourceRecordSet: &newRecordSet,
				},
				{
					Action:            aws.String("DELETE"),
					ResourceRecordSet: recordSet,
				},
			},
		},
		HostedZoneId: &hostedZoneID,
	})
	if err != nil {
		return err
	}

	logger.WithFields(log.Fields{
		"route53-dns-value":      dnsName,
		"route53-hosted-zone-id": hostedZoneID,
	}).Debugf("AWS route53 record update response: %s", prettyRoute53Response(resp))

	return nil
}

func (a *Client) deleteCNAME(hostedZoneID, dnsName string, logger log.FieldLogger) error {
	recordSets, err := a.getRecordSetsForDNS(hostedZoneID, dnsName, logger)
	if err != nil {
		return errors.Wrapf(err, "failed to get record sets for dns name %s", dnsName)
	}

	var changes []*route53.Change
	for _, recordSet := range recordSets {
		changes = append(changes, &route53.Change{
			Action:            aws.String("DELETE"),
			ResourceRecordSet: recordSet,
		})
	}
	if len(changes) == 0 {
		logger.Warn("Unable to find any DNS records; skipping...")
		return nil
	}
	if len(recordSets) != 1 {
		return errors.Errorf("expected exactly 1 resource record, but found %d", len(changes))
	}

	resp, err := a.Service().route53.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  &route53.ChangeBatch{Changes: changes},
		HostedZoneId: &hostedZoneID,
	})
	if err != nil {
		return err
	}

	logger.WithFields(log.Fields{
		"route53-records-deleted": len(changes),
		"route53-dns-value":       dnsName,
		"route53-hosted-zone-id":  hostedZoneID,
	}).Debugf("AWS route53 delete response: %s", prettyRoute53Response(resp))

	return nil
}

// getRecordSetsForDNS returns up to 10 record sets for a given DNS name and
// relies on API payload ordering behavior to function correctly. Why is this
// the case, you ask? Well...
// "Stay Awhile and Listen" - Deckard Cain
//
// Route53 records can't be filtered by the API based on the DNS value. Instead,
// they offer the ability to return records starting at a given DNS value. We
// use this and rely on the ordering always being alphabetical so if there are
// any duplicate records for a given domain then they will be immediately after
// the returned result. The returned payload also contains an indicator if there
// are more records to parse through. We no longer fetch these extra pages as
// this doesn't scale and leads to rate limiting issues. As such, this should
// only be called when expecting less than 10 records for a given DNS value and
// we will cross our fingers that the records will always be ordered correctly.
func (a *Client) getRecordSetsForDNS(hostedZoneID, dnsName string, logger log.FieldLogger) ([]*route53.ResourceRecordSet, error) {
	var recordSets []*route53.ResourceRecordSet
	recordList, err := a.Service().route53.ListResourceRecordSets(
		&route53.ListResourceRecordSetsInput{
			HostedZoneId:    &hostedZoneID,
			StartRecordName: &dnsName,
			MaxItems:        aws.String("10"),
		})
	if err != nil {
		return nil, errors.Wrap(err, "failed to list resource records")
	}

	for _, recordSet := range recordList.ResourceRecordSets {
		if strings.TrimRight(*recordSet.Name, ".") == dnsName {
			recordSets = append(recordSets, recordSet)
		}
	}

	if len(recordSets) >= 10 {
		return recordSets, errors.New("max record set (10) reached for the given DNS value; results are probably incomplete")
	}

	return recordSets, nil
}

// getHostedZoneIDWithTag returns R53 hosted zone ID for a given tag
func (a *Client) getHostedZoneIDWithTag(tag Tag) (string, error) {
	zones, err := a.getHostedZonesWithTag(tag, true)
	if err != nil {
		return "", err
	}
	if len(zones) == 0 {
		return "", errors.Errorf("no hosted zone ID associated with tag: %s", tag.String())
	}
	return parseHostedZoneResourceID(zones[0])
}

// GetHostedZonesWithTag returns R53 hosted zone for a given tag
func (a *Client) GetHostedZonesWithTag(tag Tag) ([]*route53.HostedZone, error) {
	zones, err := a.getHostedZonesWithTag(tag, false)
	if err != nil {
		return nil, err
	}
	return zones, nil
}

func (a *Client) getHostedZonesWithTag(tag Tag, firstOnly bool) ([]*route53.HostedZone, error) {
	var zones []*route53.HostedZone
	var next *string

	for {
		zoneList, err := a.Service().route53.ListHostedZones(&route53.ListHostedZonesInput{Marker: next})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to list all hosted zones")
		}

		for _, zone := range zoneList.HostedZones {
			id, err := parseHostedZoneResourceID(zone)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to parse hosted zone ID: %s", zone.String())
			}

			tagList, err := a.Service().route53.ListTagsForResource(&route53.ListTagsForResourceInput{
				ResourceId:   aws.String(id),
				ResourceType: aws.String(hostedZoneResourceType),
			})
			if err != nil {
				return nil, errors.Wrap(err, "failed to get tag list for hosted zone")
			}

			for _, resourceTag := range tagList.ResourceTagSet.Tags {
				if tag.Compare(resourceTag) {
					zones = append(zones, zone)
					break
				}
			}
			if firstOnly && len(zones) > 0 {
				return zones, nil
			}
		}

		if zoneList.Marker == nil || *zoneList.Marker == "" {
			break
		}
		next = zoneList.Marker
	}

	return zones, nil
}

func prettyRoute53Response(resp *route53.ChangeResourceRecordSetsOutput) string {
	prettyResp, err := json.Marshal(resp)
	if err != nil {
		return strings.Replace(resp.String(), "\n", " ", -1)
	}

	return string(prettyResp)
}

// parseHostedZoneResourceID removes prefix from hosted zone ID.
func parseHostedZoneResourceID(hostedZone *route53.HostedZone) (string, error) {
	id := strings.TrimLeft(*hostedZone.Id, hostedZonePrefix)
	if len(id) < hostedZoneIDLength {
		return "", errors.Errorf("invalid hosted zone ID: %s", id)
	}
	return id, nil
}

// Tag is a package specific tag with convenient methods for interacting with AWS Route53 resource tags.
type Tag struct {
	Key   string
	Value string
}

// Compare a package specific tag with a AWS Route53 resource tag.
func (t *Tag) Compare(tag *route53.Tag) bool {
	if tag != nil {
		if tag.Key != nil && *tag.Key == trimTagPrefix(t.Key) {
			if tag.Value != nil && len(*tag.Value) > 0 {
				if *tag.Value == t.Value {
					return true
				}
				return false
			}
			return true
		}
	}
	return false
}

// String prints tag's key/value.
func (t *Tag) String() string {
	return fmt.Sprintf("%s:%s", t.Key, t.Value)
}

// trimTrailingDomainDot is used to trim the trailing dot returned on route53
// hosted zone domain names.
func trimTrailingDomainDot(domain string) string {
	return strings.TrimRight(domain, ".")
}

func getString(str *string) string {
	if str != nil {
		return *str
	}
	return ""
}
