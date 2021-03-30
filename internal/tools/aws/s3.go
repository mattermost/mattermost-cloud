// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"fmt"
	"math"
	"strings"

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

// S3LargeCopy uses the "Upload Part - Copy API" from AWS to copy
// srcBucketName/srcBucketKey to destBucketName/destBucketKey in the
// case that the file being copied may be greater than 5GB in size
func (a *Client) S3LargeCopy(srcBucketName, srcBucketKey, destBucketName, destBucketKey *string) error {
	request, response := a.service.s3.CreateMultipartUploadRequest(
		&s3.CreateMultipartUploadInput{
			Bucket: destBucketName,
			Key:    destBucketKey,
		})
	err := request.Send()
	if err != nil {
		return err
	}

	uploadID := response.UploadId
	copySource := fmt.Sprintf("%s/%s", *srcBucketName, *srcBucketKey)

	objectMetadata, err := a.service.s3.HeadObject(
		&s3.HeadObjectInput{
			Bucket: srcBucketName,
			Key:    srcBucketKey,
		})
	if err != nil {
		return errors.Wrapf(err, "failed to get object metadata for %s/%s", *srcBucketName, *srcBucketKey)
	}

	objectSize := *objectMetadata.ContentLength
	var (
		partSize     int64 = 5 * 1024 * 1024 // 5 MB parts
		bytePosition int64 = 0
		partNum      int64 = 1
	)
	completedParts := []*s3.CompletedPart{}
	for ; bytePosition < objectSize; partNum++ {
		// The last part might be smaller than partSize, so check to make sure
		// that lastByte isn't beyond the end of the object.
		lastByte := int(math.Min(float64(bytePosition+partSize-1), float64(objectSize-1)))
		bytesRange := fmt.Sprintf("bytes=%d-%d", bytePosition, lastByte)

		resp, err := a.service.s3.UploadPartCopy(
			&s3.UploadPartCopyInput{
				Bucket:          destBucketName,
				CopySource:      &copySource,
				CopySourceRange: &bytesRange,
				Key:             destBucketKey,
				PartNumber:      &partNum,
				UploadId:        uploadID,
			})
		if err != nil {
			return errors.Wrapf(err, "failed to upload part %d", partNum)
		}
		bytePosition += partSize
		partNumber := partNum // copy this because AWS wants a pointer

		// for some reason the ETag comes back from AWS surrounded with quotes???
		etag := strings.TrimPrefix(strings.TrimSuffix(*resp.CopyPartResult.ETag, "\""), "\"")
		completedParts = append(completedParts,
			&s3.CompletedPart{
				ETag:       &etag,
				PartNumber: &partNumber,
			})
	}

	_, err = a.service.s3.CompleteMultipartUpload(&s3.CompleteMultipartUploadInput{
		Bucket: destBucketName,
		Key:    destBucketKey,
		MultipartUpload: &s3.CompletedMultipartUpload{
			Parts: completedParts,
		},
		UploadId: uploadID,
	})
	return err
}
