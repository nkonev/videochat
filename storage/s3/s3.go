package s3

import "github.com/minio/minio-go/v7"

type InternalMinioClient struct {
	*minio.Client
}

type PublicMinioClient struct {
	*minio.Client
}
