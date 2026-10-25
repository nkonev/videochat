package s3client

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/minio/minio-go/v7"
	"nkonev.name/storage/logger"
)

type InternalMinioClient struct {
	*minio.Client
}

func (c *InternalMinioClient) FileExists(ctx context.Context, bucket, key string) (bool, *minio.ObjectInfo, error) {
	objectInfo, err := c.Client.StatObject(ctx, bucket, key, minio.StatObjectOptions{})
	if err != nil {
		if errTyped, ok := err.(minio.ErrorResponse); ok {
			if errTyped.Code == "NoSuchKey" {
				return false, nil, nil
			}
		}
		return false, nil, err
	}
	return true, &objectInfo, err
}

func NewS3Wrapper(cl *s3.Client, lg *logger.Logger) *S3Wrapper {
	return &S3Wrapper{
		Client: cl,
		lgr:    lg,
	}
}

type S3Wrapper struct {
	*s3.Client
	lgr *logger.Logger
}

func (c *S3Wrapper) FileExists(ctx context.Context, bucket, key string) (bool, *s3.HeadObjectOutput, error) {
	objectInfo, err := c.Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})

	if err != nil {
		if IsS3NotFoundError(err) {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, objectInfo, err
}

func IsS3NotFoundError(err error) bool {
	if err != nil {
		var apiError smithy.APIError
		if errors.As(err, &apiError) {
			switch apiError.(type) {
			case *types.NotFound:
				return true
			}
		}
	}

	return false
}

func (c *S3Wrapper) EnsureBucket(ctx context.Context, bucketName string) error {
	// https://docs.aws.amazon.com/sdk-for-go/v2/developer-guide/go_s3_code_examples.html HeadBucket
	_, err := c.HeadBucket(ctx, &s3.HeadBucketInput{
		Bucket: aws.String(bucketName),
	})

	exists := true
	if IsS3NotFoundError(err) {
		exists = false
		err = nil
	} else if err != nil {
		return err
	}

	if exists {
		c.lgr.Infof("Bucket '%s' is already present", bucketName)
		return nil
	}

	_, err = c.CreateBucket(ctx, &s3.CreateBucketInput{
		Bucket: aws.String(bucketName),
	})
	if err != nil {
		c.lgr.Errorf("Error during creating bucket '%s'", bucketName)
		return err
	}

	c.lgr.Infof("Successfully created bucket '%s'", bucketName)
	return nil
}
