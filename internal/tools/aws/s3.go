package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
)

func (a *Client) s3EnsureBucketCreated(bucketName string) error {
	svc := s3.New(session.New())

	_, err := svc.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
		ACL:    aws.String("private"),
	})
	if err != nil {
		return errors.Wrap(err, "unable to create bucket")
	}

	_, err = svc.PutPublicAccessBlock(&s3.PutPublicAccessBlockInput{
		Bucket: aws.String(bucketName),
		PublicAccessBlockConfiguration: &s3.PublicAccessBlockConfiguration{
			BlockPublicAcls:       aws.Bool(true),
			BlockPublicPolicy:     aws.Bool(true),
			IgnorePublicAcls:      aws.Bool(true),
			RestrictPublicBuckets: aws.Bool(true),
		},
	})
	if err != nil {
		return errors.Wrap(err, "unable to block public bucket access")
	}

	return nil
}

func (a *Client) s3EnsureBucketDeleted(bucketName string) error {
	svc := s3.New(session.New())

	// AWS forces S3 buckets to be emptry before they can be deleted.
	iter := s3manager.NewDeleteListIterator(svc, &s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
	})

	err := s3manager.NewBatchDeleteWithClient(svc).Delete(aws.BackgroundContext(), iter)
	if err != nil {
		return errors.Wrap(err, "unable to delete bucket objects")
	}

	_, err = svc.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return errors.Wrap(err, "unable to delete bucket")
	}

	return nil
}
