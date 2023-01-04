// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.
//

package aws

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	s3ACLPrivate = "private"
)

func (a *Client) s3EnsureBucketCreated(bucketName string, logger log.FieldLogger) error {
	ctx := context.TODO()
	_, err := a.Service().s3.CreateBucket(
		ctx,
		&s3.CreateBucketInput{
			Bucket: aws.String(bucketName),
			ACL:    s3ACLPrivate,
		})
	if err != nil {
		return errors.Wrap(err, "unable to create bucket")
	}

	_, err = a.Service().s3.PutPublicAccessBlock(
		ctx,
		&s3.PutPublicAccessBlockInput{
			Bucket: aws.String(bucketName),
			PublicAccessBlockConfiguration: &types.PublicAccessBlockConfiguration{
				BlockPublicAcls:       true,
				BlockPublicPolicy:     true,
				IgnorePublicAcls:      true,
				RestrictPublicBuckets: true,
			},
		})
	if err != nil {
		return errors.Wrap(err, "unable to block public bucket access")
	}

	_, err = a.Service().s3.PutBucketEncryption(
		ctx,
		&s3.PutBucketEncryptionInput{
			Bucket: aws.String(bucketName),
			ServerSideEncryptionConfiguration: &types.ServerSideEncryptionConfiguration{
				Rules: []types.ServerSideEncryptionRule{
					{
						ApplyServerSideEncryptionByDefault: &types.ServerSideEncryptionByDefault{
							SSEAlgorithm: types.ServerSideEncryptionAes256,
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

// S3BatchDelete delete objects from a bucket in batches
func (a *Client) S3BatchDelete(bucketName string, prefix *string) error {
	ctx := context.TODO()
	paginator := s3.NewListObjectsV2Paginator(
		a.service.s3,
		&s3.ListObjectsV2Input{
			Bucket:  &bucketName,
			MaxKeys: 1000, // The maximum number of objects we can retrieve on a single request
			Prefix:  prefix,
		},
	)

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return errors.Wrap(err, "couldn't get bucket page")
		}

		// Ensure we have a page
		if page == nil {
			break
		}

		var objectIDs []types.ObjectIdentifier
		for _, obj := range page.Contents {
			objectIDs = append(objectIDs, types.ObjectIdentifier{
				Key: obj.Key,
			})
		}

		// Ensure we have objects
		if len(objectIDs) == 0 {
			a.logger.Warnf("received empty page while emptying bucket %s, assuming finished", bucketName)
			break
		}

		_, err = a.Service().s3.DeleteObjects(
			ctx,
			&s3.DeleteObjectsInput{
				Bucket: &bucketName,
				Delete: &types.Delete{
					Objects: objectIDs,
				},
			},
		)
		if err != nil {
			return errors.Wrap(err, "couldn't delete objects from bucket")
		}
	}
	return nil
}

// S3EnsureBucketDeleted is used to check if S3 bucket exists, clean it and delete it.
func (a *Client) S3EnsureBucketDeleted(bucketName string, logger log.FieldLogger) error {
	ctx := context.TODO()
	// First check if bucket still exists. There isn't a "GetBucket" so we will
	// try to get the bucket policy instead.
	_, err := a.Service().s3.HeadBucket(
		ctx,
		&s3.HeadBucketInput{
			Bucket: aws.String(bucketName),
		})
	if err != nil {
		var awsNotFound *types.NotFound
		if errors.As(err, &awsNotFound) {
			logger.WithField("s3-bucket-name", bucketName).Warn("AWS S3 bucket could not be found; assuming already deleted")
			return nil
		}
		logger.WithField("s3-bucket-name", bucketName).WithError(err).Warn("Could not determine if S3 bucket exists")
	}

	if err := a.S3BatchDelete(bucketName, nil); err != nil {
		return errors.Wrap(err, "can't empty bucket contents")
	}

	_, err = a.Service().s3.DeleteBucket(
		ctx,
		&s3.DeleteBucketInput{
			Bucket: aws.String(bucketName),
		})
	if err != nil {
		return errors.Wrap(err, "unable to delete bucket")
	}

	return nil
}

// S3EnsureBucketDirectoryDeleted is used to ensure that a bucket directory is deleted.
func (a *Client) S3EnsureBucketDirectoryDeleted(bucketName, directory string, _ log.FieldLogger) error {
	return a.S3BatchDelete(bucketName, &directory)
}

// S3LargeCopy uses the "Upload Part - Copy API" from AWS to copy
// srcBucketName/srcBucketKey to destBucketName/destBucketKey in the
// case that the file being copied may be greater than 5GB in size
func (a *Client) S3LargeCopy(srcBucketName, srcBucketKey, destBucketName, destBucketKey *string) error {
	ctx := context.TODO()
	response, err := a.service.s3.CreateMultipartUpload(
		ctx,
		&s3.CreateMultipartUploadInput{
			Bucket: destBucketName,
			Key:    destBucketKey,
		})

	uploadID := response.UploadId
	copySource := fmt.Sprintf("%s/%s", *srcBucketName, *srcBucketKey)

	objectMetadata, err := a.service.s3.HeadObject(
		ctx,
		&s3.HeadObjectInput{
			Bucket: srcBucketName,
			Key:    srcBucketKey,
		})
	if err != nil {
		return errors.Wrapf(err, "failed to get object metadata for %s/%s", *srcBucketName, *srcBucketKey)
	}

	objectSize := objectMetadata.ContentLength
	var (
		partSize     int64 = 256 * 1024 * 1024 // 256 MB parts
		bytePosition int64 = 0
		partNum      int32 = 1
	)
	completedParts := []types.CompletedPart{}
	for ; bytePosition < objectSize; partNum++ {
		// The last part might be smaller than partSize, so check to make sure
		// that lastByte isn't beyond the end of the object.
		lastByte := int(math.Min(float64(bytePosition+partSize-1), float64(objectSize-1)))
		bytesRange := fmt.Sprintf("bytes=%d-%d", bytePosition, lastByte)

		resp, err := a.service.s3.UploadPartCopy(
			ctx,
			&s3.UploadPartCopyInput{
				Bucket:          destBucketName,
				CopySource:      &copySource,
				CopySourceRange: &bytesRange,
				Key:             destBucketKey,
				PartNumber:      partNum,
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
			types.CompletedPart{
				ETag:       &etag,
				PartNumber: partNumber,
			})
	}

	_, err = a.service.s3.CompleteMultipartUpload(
		ctx,
		&s3.CompleteMultipartUploadInput{
			Bucket: destBucketName,
			Key:    destBucketKey,
			MultipartUpload: &types.CompletedMultipartUpload{
				Parts: completedParts,
			},
			UploadId: uploadID,
		})

	return err
}

// S3EnsureObjectDeleted is used to ensure that the file is deleted.
func (a *Client) S3EnsureObjectDeleted(bucketName, path string) error {
	_, err := a.Service().s3.DeleteObject(
		context.TODO(),
		&s3.DeleteObjectInput{
			Bucket: aws.String(bucketName),
			Key:    aws.String(path),
		})
	if err != nil {
		return errors.Wrap(err, "failed to delete object")
	}

	return nil
}
