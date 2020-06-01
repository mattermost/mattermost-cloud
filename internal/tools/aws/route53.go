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

// CreatePublicCNAME creates a record in Route53 for a public domain name.
func (a *Client) CreatePublicCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error {
	id, err := a.getHostedZoneIDWithTag(Tag{
		Key:   DefaultCloudDNSTagKey,
		Value: DefaultPublicCloudDNSTagValue,
	}, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to create a public CNAME: %s", dnsName)
	}

	return a.createCNAME(id, dnsName, dnsEndpoints, logger)
}

// CreatePrivateCNAME creates a record in Route53 for a private domain name.
func (a *Client) CreatePrivateCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error {
	id, err := a.getHostedZoneIDWithTag(Tag{
		Key:   DefaultCloudDNSTagKey,
		Value: DefaultPrivateCloudDNSTagValue,
	}, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to create a private CNAME: %s", dnsName)
	}

	return a.createCNAME(id, dnsName, dnsEndpoints, logger)
}

// GetPrivateZoneDomainName gets the private Route53 domain name.
func (a *Client) GetPrivateZoneDomainName(logger log.FieldLogger) (string, error) {
	id, err := a.GetPrivateZoneID(logger)
	if err != nil {
		return "", err
	}
	return a.getZoneDNS(id, logger)
}

// GetPrivateZoneID returns the R53 hosted zone ID for a given tag or if not provided for the default one
func (a *Client) GetPrivateZoneID(logger log.FieldLogger, tags ...Tag) (string, error) {
	if len(tags) > 1 {
		return "", errors.New("does not support more than 1 tag yet")
	}
	tag := Tag{
		Key:   DefaultCloudDNSTagKey,
		Value: DefaultPrivateCloudDNSTagValue,
	}
	if len(tags) > 0 {
		tag = tags[0]
	}
	id, err := a.getHostedZoneIDWithTag(tag, logger)
	if err != nil {
		return "", errors.Wrap(err, "unable to get private domain name")
	}
	return id, nil
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

func (a *Client) createCNAME(hostedZoneID, dnsName string, dnsEndpoints []string, logger log.FieldLogger) error {
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
						SetIdentifier:   aws.String(dnsName),
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

// IsProvisionedPrivateCNAME returns true if a record has been
// registered for the given CNAME (full FQDN required as input)
func (a *Client) IsProvisionedPrivateCNAME(dnsName string, logger log.FieldLogger) bool {
	id, err := a.getHostedZoneIDWithTag(Tag{
		Key:   DefaultCloudDNSTagKey,
		Value: DefaultPrivateCloudDNSTagValue,
	}, logger)

	if err != nil {
		logger.WithError(err).Debugf("couldn't look up zone ID for DNS name %s", dnsName)
		return false
	}

	return a.isProvisionedCNAME(id, dnsName, logger)
}

// DeletePublicCNAME deletes a AWS route53 record for a public domain name.
func (a *Client) DeletePublicCNAME(dnsName string, logger log.FieldLogger) error {
	id, err := a.getHostedZoneIDWithTag(Tag{
		Key:   DefaultCloudDNSTagKey,
		Value: DefaultPublicCloudDNSTagValue,
	}, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to delete a public CNAME: %s", dnsName)
	}

	return a.deleteCNAME(id, dnsName, logger)
}

// DeletePrivateCNAME deletes a AWS route53 record for a private domain name.
func (a *Client) DeletePrivateCNAME(dnsName string, logger log.FieldLogger) error {
	id, err := a.getHostedZoneIDWithTag(Tag{
		Key:   DefaultCloudDNSTagKey,
		Value: DefaultPrivateCloudDNSTagValue,
	}, logger)
	if err != nil {
		return errors.Wrapf(err, "unable to delete a private CNAME: %s", dnsName)
	}

	return a.deleteCNAME(id, dnsName, logger)
}

func (a *Client) isProvisionedCNAME(hostedZoneID, dnsName string, logger log.FieldLogger) bool {
	nextRecordName := dnsName
	for {
		recordList, err := a.Service().route53.ListResourceRecordSets(
			&route53.ListResourceRecordSetsInput{
				HostedZoneId:    &hostedZoneID,
				StartRecordName: &nextRecordName,
			})

		if err != nil {
			logger.WithError(err).Debugf("couldn't find record list for %s", dnsName)
			return false
		}

		for _, recordSet := range recordList.ResourceRecordSets {
			if recordSet.Name != nil && dnsName == strings.TrimRight(*recordSet.Name, ".") {
				return true
			}
		}

		if !*recordList.IsTruncated {
			break
		}

		nextRecordName = *recordList.NextRecordName
		logger.Debugf("DNS query found more than one page of records; running another query with record-name=%s", nextRecordName)
	}

	return false
}

func (a *Client) deleteCNAME(hostedZoneID, dnsName string, logger log.FieldLogger) error {
	nextRecordName := dnsName
	var recordSets []*route53.ResourceRecordSet
	for {
		recordList, err := a.Service().route53.ListResourceRecordSets(
			&route53.ListResourceRecordSetsInput{
				HostedZoneId:    &hostedZoneID,
				StartRecordName: &nextRecordName,
			})
		if err != nil {
			return err
		}

		recordSets = append(recordSets, recordList.ResourceRecordSets...)

		if !*recordList.IsTruncated {
			break
		}

		// Too many records were received. We need to keep going.
		nextRecordName = *recordList.NextRecordName
		logger.Debugf("DNS query found more than one page of records; running another query with record-name=%s", nextRecordName)
	}

	var changes []*route53.Change
	for _, recordSet := range recordSets {
		if strings.Trim(*recordSet.Name, ".") == dnsName {
			changes = append(changes, &route53.Change{
				Action:            aws.String("DELETE"),
				ResourceRecordSet: recordSet,
			})
		}
	}
	if len(changes) == 0 {
		logger.Warn("Unable to find any DNS records; skipping...")
		return nil
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

// getHostedZoneIDWithTag returns R53 hosted zone ID for a given tag
func (a *Client) getHostedZoneIDWithTag(tag Tag, logger log.FieldLogger) (string, error) {
	var next *string
	for {
		zoneList, err := a.Service().route53.ListHostedZones(&route53.ListHostedZonesInput{Marker: next})
		if err != nil {
			return "", errors.Wrapf(err, "listing hosted all zones")
		}

		for _, zone := range zoneList.HostedZones {
			id, err := parseHostedZoneResourceID(zone)
			if err != nil {
				return "", errors.Wrapf(err, "when parsing hosted zone: %s", zone.String())
			}

			tagList, err := a.Service().route53.ListTagsForResource(&route53.ListTagsForResourceInput{
				ResourceId:   aws.String(id),
				ResourceType: aws.String(hostedZoneResourceType),
			})
			if err != nil {
				return "", err
			}

			for _, resourceTag := range tagList.ResourceTagSet.Tags {
				if tag.Compare(resourceTag) {
					return id, nil
				}
			}
		}

		if zoneList.Marker == nil || *zoneList.Marker == "" {
			break
		}
		next = zoneList.Marker
	}

	return "", errors.Errorf("no hosted zone ID associated with tag: %s", tag.String())
}

func prettyRoute53Response(resp *route53.ChangeResourceRecordSetsOutput) string {
	prettyResp, err := json.Marshal(resp)
	if err != nil {
		return strings.Replace(resp.String(), "\n", " ", -1)
	}

	return string(prettyResp)
}

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
