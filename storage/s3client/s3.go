package s3client

import (
	"context"
	"errors"

	s3v2 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/minio/minio-go/v7"
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

type S3Wrapper struct {
	*s3v2.Client
}

func (c *S3Wrapper) FileExists(ctx context.Context, bucket, key string) (bool, *s3v2.HeadObjectOutput, error) {
	objectInfo, err := c.Client.HeadObject(ctx, &s3v2.HeadObjectInput{
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
