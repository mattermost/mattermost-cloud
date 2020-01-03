package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/pkg/errors"
)

// GetCertificateSummaryByTag returns the certificate summary associated with a valid tag key and value in AWS.
func (a *Client) GetCertificateSummaryByTag(key, value string) (*acm.CertificateSummary, error) {
	svc, err := a.api.getACMClient()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get ACM client")
	}

	key = trimTagPrefix(key)
	tag := acm.Tag{Key: &key, Value: &value}

	var next *string
	for {
		out, err := a.api.listCertificates(svc, &acm.ListCertificatesInput{
			NextToken: next,
		})
		if err != nil {
			return nil, errors.Wrap(err, "error fetching certificates")
		}

		for _, cert := range out.CertificateSummaryList {
			list, err := a.api.listTagsForCertificate(svc, &acm.ListTagsForCertificateInput{CertificateArn: cert.CertificateArn})
			if err != nil {
				return nil, errors.Wrapf(err, "error listing tags for certificate %s", *cert.CertificateArn)
			}
			for _, v := range list.Tags {
				if v.Key != nil && *v.Key == *tag.Key {
					if v.Value != nil {
						if *v.Value == *tag.Value {
							return cert, nil
						}
						continue
					}
					return cert, nil
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

func (api *apiInterface) getACMClient() (*acm.ACM, error) {
	sess, err := session.NewSession()
	if err != nil {
		return nil, err
	}

	return acm.New(sess, &aws.Config{
		Region: aws.String(DefaultAWSRegion),
	}), nil
}

func (api *apiInterface) listCertificates(svc *acm.ACM, input *acm.ListCertificatesInput) (*acm.ListCertificatesOutput, error) {
	return svc.ListCertificates(input)
}

func (api *apiInterface) listTagsForCertificate(svc *acm.ACM, input *acm.ListTagsForCertificateInput) (*acm.ListTagsForCertificateOutput, error) {
	return svc.ListTagsForCertificate(input)
}
