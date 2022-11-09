// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/acm"
	acmTypes "github.com/aws/aws-sdk-go-v2/service/acm/types"
	"github.com/mattermost/mattermost-cloud/model"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// GetCertificateSummaryByTag returns the certificate summary associated with a valid tag key and value in AWS.
func (a *Client) GetCertificateSummaryByTag(key, value string, logger log.FieldLogger) (*model.Certificate, error) {
	ctx := context.TODO()
	key = trimTagPrefix(key)
	tag := acmTypes.Tag{Key: &key, Value: &value}

	var next *string
	for {
		out, err := a.Service().acm.ListCertificates(ctx, &acm.ListCertificatesInput{
			NextToken: next,
		})
		if err != nil {
			return nil, errors.Wrap(err, "error fetching certificates")
		}

		for _, cert := range out.CertificateSummaryList {
			list, err := a.Service().acm.ListTagsForCertificate(ctx, &acm.ListTagsForCertificateInput{CertificateArn: cert.CertificateArn})
			if err != nil {
				return nil, errors.Wrapf(err, "error listing tags for certificate %s", *cert.CertificateArn)
			}
			for _, v := range list.Tags {
				if v.Key != nil && *v.Key == *tag.Key {
					if v.Value != nil {
						if *v.Value == *tag.Value {
							return newCertificateFromACMCertificateSummary(cert), nil
						}
						continue
					}
					return newCertificateFromACMCertificateSummary(cert), nil
				}
			}
		}

		if out.NextToken == nil || *out.NextToken == "" {
			break
		}
		next = out.NextToken
	}

	return nil, errors.Errorf("no certificate was found under tag:%s:%s", *tag.Key, *tag.Value)
}
