package s3

import (
	"context"
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
