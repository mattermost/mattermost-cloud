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

// GetHostedZoneIDWithTag ...
func (a *Client) getHostedZoneIDWithTag(tag Tag) (string, error) {
	svc, err := a.api.getRoute53Client()
	if err != nil {
		return "", err
	}

	var next *string
	for {
		zoneList, err := svc.ListHostedZones(&route53.ListHostedZonesInput{Marker: next})
		if err != nil {
			return "", errors.Wrapf(err, "listing hosted all zones")
		}

		for _, zone := range zoneList.HostedZones {
			id, err := parseHostedZoneResourceID(zone)
			if err != nil {
				return "", errors.Wrapf(err, "when parsing hosted zone: %s", zone.String())
			}

			tagList, err := svc.ListTagsForResource(&route53.ListTagsForResourceInput{
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

// CreatePublicCNAME ..
func (a *Client) CreatePublicCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error {
	id, err := a.getHostedZoneIDWithTag(Tag{
		Key:   DefaultCloudDNSTagKey,
		Value: DefaultPublicCloudDNSTagValue,
	})
	if err != nil {
		return errors.Wrapf(err, "creating a public CNAME")
	}

	return a.CreateCNAME(id, dnsName, dnsEndpoints, logger)
}

// CreatePrivateCNAME ...
func (a *Client) CreatePrivateCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error {
	id, err := a.getHostedZoneIDWithTag(Tag{
		Key:   DefaultCloudDNSTagKey,
		Value: DefaultPrivateCloudDNSTagValue,
	})
	if err != nil {
		return errors.Wrapf(err, "creating a private CNAME")
	}

	return a.CreateCNAME(id, dnsName, dnsEndpoints, logger)
}

// CreateCNAME creates an AWS route53 CNAME record.
func (a *Client) CreateCNAME(hostedZoneID, dnsName string, dnsEndpoints []string, logger log.FieldLogger) error {
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

// DeletePublicCNAME ..
func (a *Client) DeletePublicCNAME(dnsName string, logger log.FieldLogger) error {
	id, err := a.getHostedZoneIDWithTag(Tag{
		Key:   DefaultCloudDNSTagKey,
		Value: DefaultPublicCloudDNSTagValue,
	})
	if err != nil {
		return errors.Wrapf(err, "creating a private CNAME")
	}

	return a.DeleteCNAME(id, dnsName, logger)
}

// DeletePrivateCNAME ..
func (a *Client) DeletePrivateCNAME(dnsName string, logger log.FieldLogger) error {
	id, err := a.getHostedZoneIDWithTag(Tag{
		Key:   DefaultCloudDNSTagKey,
		Value: DefaultPrivateCloudDNSTagValue,
	})
	if err != nil {
		return errors.Wrapf(err, "creating a private CNAME")
	}

	return a.DeleteCNAME(id, dnsName, logger)
}

// DeleteCNAME deletes an AWS route53 CNAME record.
func (a *Client) DeleteCNAME(hostedZoneID, dnsName string, logger log.FieldLogger) error {
	svc, err := a.api.getRoute53Client()
	if err != nil {
		return err
	}

	if len(dnsName) >= 64 {
		logger.Warnf("DNS name too long, skipping since it could never have been created record-name=%s", dnsName)
		return nil
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

// Tag ...
type Tag struct {
	Key   string
	Value string
}

// Compare ..
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

func (t *Tag) String() string {
	return fmt.Sprintf("%s:%s", t.Key, t.Value)
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

func parseHostedZoneResourceID(hostedZone *route53.HostedZone) (string, error) {
	id := strings.TrimLeft(*hostedZone.Id, hostedZonePrefix)
	if len(id) < hostedZoneIDLength {
		return "", errors.Errorf("invalid hosted zone ID: %s", id)
	}
	return id, nil
}
