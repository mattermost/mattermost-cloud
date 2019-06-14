package aws

import (
	"errors"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
	log "github.com/sirupsen/logrus"
)

// CreateCNAME creates an AWS route53 CNAME record.
func (a *Client) CreateCNAME(dnsName string, dnsEndpoints []string, logger log.FieldLogger) error {
	if len(dnsEndpoints) == 0 {
		return errors.New("no DNS endpoints provided for route53 creation request")
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
		HostedZoneId: aws.String(a.hostedZoneID),
	}

	resp, err := a.api.changeResourceRecordSets(svc, input)
	if err != nil {
		return err
	}

	logger.Debugf("AWS route53 response: %s", resp)

	return nil
}

// DeleteCNAME deletes an AWS route53 CNAME record.
func (a *Client) DeleteCNAME(dnsName string, logger log.FieldLogger) error {
	sess, err := session.NewSession()
	if err != nil {
		return err
	}
	svc := route53.New(sess)

	nextRecordName := dnsName
	var recordsets []*route53.ResourceRecordSet
	for {
		recordList, err := a.api.listResourceRecordSets(svc,
			&route53.ListResourceRecordSetsInput{
				HostedZoneId:    &a.hostedZoneID,
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

	logger.Debugf("Attempting to delete %d DNS records", len(changes))

	input := &route53.ChangeResourceRecordSetsInput{
		ChangeBatch:  &route53.ChangeBatch{Changes: changes},
		HostedZoneId: aws.String(a.hostedZoneID),
	}
	resp, err := a.api.changeResourceRecordSets(svc, input)
	if err != nil {
		return err
	}

	logger.Debugf("AWS route53 response: %s", resp)

	return nil
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
