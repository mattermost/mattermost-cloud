package aws

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
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
	})
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
	})
	if err != nil {
		return errors.Wrapf(err, "unable to create a private CNAME: %s", dnsName)
	}

	return a.createCNAME(id, dnsName, dnsEndpoints, logger)
}

// GetPrivateZoneDomainName gets the private Route53 domain name.
func (a *Client) GetPrivateZoneDomainName(logger log.FieldLogger) (string, error) {
	id, err := a.getHostedZoneIDWithTag(Tag{
		Key:   DefaultCloudDNSTagKey,
		Value: DefaultPrivateCloudDNSTagValue,
	})
	if err != nil {
		return "", errors.Wrap(err, "unable to get private domain name")
	}

	return a.getZoneDNS(id, logger)
}

func (a *Client) getZoneDNS(hostedZoneID string, logger log.FieldLogger) (string, error) {
	svc, err := a.api.getRoute53Client()
	if err != nil {
		return "", err
	}

	out, err := svc.GetHostedZone(&route53.GetHostedZoneInput{
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

	svc, err := a.api.getRoute53Client()
	if err != nil {
		return err
	}

	var resourceRecords []*route53.ResourceRecord
	for _, endpoint := range dnsEndpoints {
		resourceRecords = append(resourceRecords, &route53.ResourceRecord{
			Value: aws.String(endpoint),
		})
	}

	input := &route53.ChangeResourceRecordSetsInput{
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
	}

	resp, err := a.api.changeResourceRecordSets(svc, input)
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

// DeletePublicCNAME deletes a AWS route53 record for a public domain name.
func (a *Client) DeletePublicCNAME(dnsName string, logger log.FieldLogger) error {
	id, err := a.getHostedZoneIDWithTag(Tag{
		Key:   DefaultCloudDNSTagKey,
		Value: DefaultPublicCloudDNSTagValue,
	})
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
	})
	if err != nil {
		return errors.Wrapf(err, "unable to delete a private CNAME: %s", dnsName)
	}

	return a.deleteCNAME(id, dnsName, logger)
}

func (a *Client) deleteCNAME(hostedZoneID, dnsName string, logger log.FieldLogger) error {
	svc, err := a.api.getRoute53Client()
	if err != nil {
		return err
	}

	nextRecordName := dnsName
	var recordsets []*route53.ResourceRecordSet
	for {
		recordList, err := a.api.listResourceRecordSets(svc,
			&route53.ListResourceRecordSetsInput{
				HostedZoneId:    &hostedZoneID,
				StartRecordName: &nextRecordName,
			})
		if err != nil {
			return err
		}

		recordsets = append(recordsets, recordList.ResourceRecordSets...)

		if !*recordList.IsTruncated {
			break
		}

		// Too many records were received. We need to keep going.
		nextRecordName = *recordList.NextRecordName
		logger.Debugf("DNS query found more than one page of records; running another query with record-name=%s", nextRecordName)
	}

	var changes []*route53.Change
	for _, recordset := range recordsets {
		if strings.Trim(*recordset.Name, ".") == dnsName {
			changes = append(changes, &route53.Change{
				Action:            aws.String("DELETE"),
				ResourceRecordSet: recordset,
			})
		}
	}
	if len(changes) == 0 {
		logger.Warn("Unable to find any DNS records; skipping...")
		return nil
	}

	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  &route53.ChangeBatch{Changes: changes},
		HostedZoneId: &hostedZoneID,
	}
	resp, err := a.api.changeResourceRecordSets(svc, input)
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

func (a *Client) getHostedZoneIDWithTag(tag Tag) (string, error) {
	svc, err := a.api.getRoute53Client()
	if err != nil {
		return "", err
	}

	var next *string
	for {
		zoneList, err := a.api.listHostedZones(svc, &route53.ListHostedZonesInput{Marker: next})
		if err != nil {
			return "", errors.Wrapf(err, "listing hosted all zones")
		}

		for _, zone := range zoneList.HostedZones {
			id, err := parseHostedZoneResourceID(zone)
			if err != nil {
				return "", errors.Wrapf(err, "when parsing hosted zone: %s", zone.String())
			}

			tagList, err := a.api.listTagsForResource(svc, &route53.ListTagsForResourceInput{
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

func (api *apiInterface) getRoute53Client() (*route53.Route53, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	return route53.New(sess), nil
}

func (api *apiInterface) changeResourceRecordSets(svc *route53.Route53, input *route53.ChangeResourceRecordSetsInput) (*route53.ChangeResourceRecordSetsOutput, error) {
	return svc.ChangeResourceRecordSets(input)
}

func (api *apiInterface) listResourceRecordSets(svc *route53.Route53, input *route53.ListResourceRecordSetsInput) (*route53.ListResourceRecordSetsOutput, error) {
	return svc.ListResourceRecordSets(input)
}

func (api *apiInterface) listHostedZones(svc *route53.Route53, input *route53.ListHostedZonesInput) (*route53.ListHostedZonesOutput, error) {
	return svc.ListHostedZones(input)
}

func (api *apiInterface) listTagsForResource(svc *route53.Route53, input *route53.ListTagsForResourceInput) (*route53.ListTagsForResourceOutput, error) {
	return svc.ListTagsForResource(input)
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
