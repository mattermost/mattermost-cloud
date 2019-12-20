package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/acm"
	"github.com/pkg/errors"
)

// TODO(gsagula): create interface for mocking and write tests.

// GetCertificateByTag returns the certificate summary associated with a valid tag key and value in AWS.
func GetCertificateByTag(key, value string) (*acm.CertificateSummary, error) {
	key = trimTagPrefix(key)
	tag := acm.Tag{Key: &key, Value: &value}
	svc := acm.New(session.New(), &aws.Config{
		Region: aws.String(DefaultAWSRegion),
	})

	var next *string
	for {
		out, err := svc.ListCertificates(&acm.ListCertificatesInput{
			NextToken: next,
		})
		if err != nil {
			return nil, err
		}
		for _, cert := range out.CertificateSummaryList {
			ok, err := isCertificateTag(cert.CertificateArn, &tag)
			if err != nil {
				return nil, err
			}
			if ok {
				return cert, nil
			}
		}
		if out.NextToken == nil || *out.NextToken == "" {
			break
		}
		next = out.NextToken
	}

	return nil, errors.Errorf("no certificate was found under tag:%s:%s", *tag.Key, *tag.Value)
}

func isCertificateTag(arn *string, tag *acm.Tag) (bool, error) {
	if tag == nil || tag.Key == nil || *tag.Key == "" {
		return false, errors.New("tag key cannot be empty")
	}
	svc := acm.New(session.New(), &aws.Config{
		Region: aws.String(DefaultAWSRegion),
	})

	list, err := svc.ListTagsForCertificate(&acm.ListTagsForCertificateInput{CertificateArn: arn})
	if err != nil {
		return false, err
	}

	for _, v := range list.Tags {
		if *v.Key == *tag.Key {
			if tag.Value != nil {
				return *v.Value == *tag.Value, nil
			}
			return true, nil
		}
	}

	return false, nil
}
