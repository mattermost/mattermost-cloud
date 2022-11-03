// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
// 

package awsv2

import (
	"context"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go-v2/service/acm"
	"github.com/mattermost/mattermost-cloud/model"
)

// GetCertificateByTag returns the certificate summary associated with a valid tag key and value in AWS.
func (c *Client) GetCertificateByTag(ctx context.Context, key, value string) (*model.Certificate, error) {
	var next *string
	for {
		out, err := c.aws.ACM.ListCertificates(ctx, &acm.ListCertificatesInput{
			NextToken: next,
		})
		if err != nil {
			return nil, errors.Wrap(err, "error fetching certificates")
		}

		for _, cert := range out.CertificateSummaryList {
			list, err := c.aws.ACM.ListTagsForCertificate(ctx, &acm.ListTagsForCertificateInput{CertificateArn: cert.CertificateArn})
			if err != nil {
				return nil, errors.Wrapf(err, "error listing tags for certificate %s", *cert.CertificateArn)
			}
			for _, v := range list.Tags {
				if v.Key != nil && *v.Key == key {
					if v.Value != nil {
						if *v.Value == value {
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

	return nil, errors.Errorf("no certificate found for tag(%s)=%v", key, value)
}
