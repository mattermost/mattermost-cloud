// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

func (a *Client) s3EnsureBucketCreated(bucketName string, logger log.FieldLogger) error {
	_, err := a.Service().s3.CreateBucket(&s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
		ACL:    aws.String("private"),
	})
	if err != nil {
		return errors.Wrap(err, "unable to create bucket")
	}

	_, err = a.Service().s3.PutPublicAccessBlock(&s3.PutPublicAccessBlockInput{
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

	_, err = a.Service().s3.PutBucketEncryption(&s3.PutBucketEncryptionInput{
		Bucket: aws.String(bucketName),
		ServerSideEncryptionConfiguration: &s3.ServerSideEncryptionConfiguration{
			Rules: []*s3.ServerSideEncryptionRule{
				{
					ApplyServerSideEncryptionByDefault: &s3.ServerSideEncryptionByDefault{
						SSEAlgorithm: aws.String(s3.ServerSideEncryptionAes256),
					},
				},
			},
		},
	})
	if err != nil {
		return errors.Wrap(err, "unable to set bucket encryption default")
	}

	return nil
}

// S3EnsureBucketDeleted is used to check if S3 bucket exists, clean it and delete it.
func (a *Client) S3EnsureBucketDeleted(bucketName string, logger log.FieldLogger) error {
	// First check if bucket still exists. There isn't a "GetBucket" so we will
	// try to get the bucket policy instead.
	_, err := a.Service().s3.GetBucketPolicy(&s3.GetBucketPolicyInput{
		Bucket: aws.String(bucketName),
	})
	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() == s3.ErrCodeNoSuchBucket {
			logger.WithField("s3-bucket-name", bucketName).Warn("AWS S3 bucket could not be found; assuming already deleted")
			return nil
		}
	}

	// AWS forces S3 buckets to be empty before they can be deleted.
	iter := s3manager.NewDeleteListIterator(a.Service().s3, &s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
	})

	err = s3manager.NewBatchDeleteWithClient(a.Service().s3).Delete(aws.BackgroundContext(), iter)
	if err != nil {
		return errors.Wrap(err, "unable to delete bucket contents")
	}

	_, err = a.Service().s3.DeleteBucket(&s3.DeleteBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		return errors.Wrap(err, "unable to delete bucket")
	}

	return nil
}

// S3EnsureBucketDirectoryDeleted is used to ensure that a bucket directory is
// deleted.
func (a *Client) S3EnsureBucketDirectoryDeleted(bucketName, directory string, logger log.FieldLogger) error {
	iter := s3manager.NewDeleteListIterator(a.Service().s3, &s3.ListObjectsInput{
		Bucket: aws.String(bucketName),
		Prefix: aws.String(directory),
	})

	err := s3manager.NewBatchDeleteWithClient(a.Service().s3).Delete(aws.BackgroundContext(), iter)
	if err != nil {
		return errors.Wrap(err, "failed to delete bucket directory")
	}

	return nil
}

// S3EnsureObjectDeleted is used to ensure that the file is deleted.
func (a *Client) S3EnsureObjectDeleted(bucketName, path string) error {
	_, err := a.Service().s3.DeleteObject(&s3.DeleteObjectInput{
		Bucket: aws.String(bucketName),
		Key:    aws.String(path),
	})
	if err != nil {
		return errors.Wrap(err, "failed to delete object")
	}

	return nil
}
