package client

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/spf13/viper"
	"nkonev.name/storage/dto"
	"nkonev.name/storage/logger"
)

type uploadParamsFromContextPresigner struct {
	delegate *v4.Signer
}

func (s *uploadParamsFromContextPresigner) PresignHTTP(
	ctx context.Context, credentials aws.Credentials, r *http.Request,
	payloadHash string, service string, region string, signingTime time.Time,
	optFns ...func(*v4.SignerOptions),
) (url string, signedHeader http.Header, err error) {
	cloned := r.Clone(ctx)

	uplid, ok := dto.UpploadIdFromContext(ctx)
	if ok {
		pn, ok := dto.PartNumberFromContext(ctx)
		if !ok {
			err = fmt.Errorf("uploadId requires partNumber as well")
			return
		}

		turl := cloned.URL
		qq := turl.Query()

		// multipart upload proxying 2/6
		qq.Del("x-id") // s3 going to return error otherwise

		// multipart upload proxying 3/6
		qq.Add("uploadId", uplid)
		qq.Add("partNumber", pn)

		turl.RawQuery = qq.Encode()

		cloned.URL = turl
	}

	return s.delegate.PresignHTTP(ctx, credentials, cloned, payloadHash, service, region, signingTime, optFns...)
}

type RequestUserMetadata struct {
}

func AddRequestUserMetadataMiddleware(stack *middleware.Stack) error {
	return stack.Build.Add(&RequestUserMetadata{}, middleware.After)
}

func (m *RequestUserMetadata) ID() string { return "RequestUserMetadata" }

func (m *RequestUserMetadata) HandleBuild(
	ctx context.Context, in middleware.BuildInput, next middleware.BuildHandler,
) (
	out middleware.BuildOutput, metadata middleware.Metadata, err error,
) {
	req, ok := in.Request.(*smithyhttp.Request)
	if !ok {
		return out, metadata, fmt.Errorf("unknown request type %T", req)
	}

	ok = dto.RequestUserMetadataFromContext(ctx)
	if ok {
		q := req.URL.Query()
		q.Set("metadata", "true")
		req.URL.RawQuery = q.Encode()
	}

	return next.HandleBuild(ctx, in)
}

func AddCacheBodyToSmthMiddleware(stack *middleware.Stack) error {
	// /home/nkonev/go/pkg/mod/github.com/aws/aws-sdk-go-v2/service/s3@v1.114.5/internal/customizations/handle_200_error.go
	return stack.Deserialize.Insert(&CacheBodyToSmth{}, "OperationDeserializer", middleware.After)
}

type CacheBodyToSmth struct{}

func (*CacheBodyToSmth) ID() string {
	return "CacheBodyToSmth"
}

type CacheBodyToStringKeyType string

const CacheBodyToStringKey CacheBodyToStringKeyType = "CacheBodyToString"

func SetCachedResponseToStringToMetadata(metadata middleware.Metadata, v string) {
	metadata.Set(CacheBodyToStringKey, v)
}

type CacheBodyToBytesKeyType string

const CacheBodyToBytesKey CacheBodyToBytesKeyType = "CacheBodyToBytes"

func SetCachedResponseToBytesToMetadata(metadata middleware.Metadata, v []byte) {
	metadata.Set(CacheBodyToBytesKey, v)
}

func GetStringCachedResponse(metadata middleware.Metadata) (string, error) {
	v := metadata.Get(CacheBodyToStringKey)
	switch typed := v.(type) {
	case string:
		return typed, nil
	default:
		return "", fmt.Errorf("unknown type: %T", typed)
	}
}

func GetBytesCachedResponse(metadata middleware.Metadata) ([]byte, error) {
	v := metadata.Get(CacheBodyToBytesKey)
	switch typed := v.(type) {
	case []byte:
		return typed, nil
	default:
		return nil, fmt.Errorf("unknown type: %T", typed)
	}
}

func (m *CacheBodyToSmth) HandleDeserialize(
	ctx context.Context, input middleware.DeserializeInput, next middleware.DeserializeHandler,
) (
	output middleware.DeserializeOutput, metadata middleware.Metadata, err error,
) {
	out, metadata, err := next.HandleDeserialize(ctx, input)
	if err != nil {
		return out, metadata, fmt.Errorf("error next.HandleDeserialize %w", err)
	}

	oks := dto.CacheResponseToStringFromContext(ctx)
	okb := dto.CacheResponseToBytesFromContext(ctx)
	if oks || okb {
		resp, ok := out.RawResponse.(*smithyhttp.Response)
		if !ok {
			return out, metadata, fmt.Errorf("unknown response type %T", resp)
		}
		origBody := resp.Body
		defer origBody.Close()

		bb, err := io.ReadAll(origBody)
		if err != nil {
			return out, metadata, fmt.Errorf("error reading response %w", err)
		}

		if oks {
			strbb := string(bb)
			SetCachedResponseToStringToMetadata(metadata, strbb)
		}
		if okb {
			SetCachedResponseToBytesToMetadata(metadata, bb)
		}

		resp.Body = io.NopCloser(bytes.NewReader(bb))
	}

	return out, metadata, err
}

// https://docs.rustfs.com/en/developer/sdk/go
// https://docs.aws.amazon.com/code-library/latest/ug/go_2_s3_code_examples.html
func ConfigureAwsS3v2(lgr *logger.Logger) (*S3Wrapper, error) {
	endpoint := viper.GetString("minio.internalEndpoint")
	accessKeyID := viper.GetString("minio.accessKeyId")
	secretAccessKey := viper.GetString("minio.secretAccessKey")
	location := viper.GetString("minio.location")
	secured := viper.GetBool("minio.secured")

	// build aws.Config
	cfg := aws.Config{
		Region:      location,
		Credentials: aws.NewCredentialsCache(credentials.NewStaticCredentialsProvider(accessKeyID, secretAccessKey, "")),
	}

	// build S3 client
	s3client := s3.NewFromConfig(cfg, func(o *s3.Options) {
		o.BaseEndpoint = &endpoint
		// RustFS uses path-style URLs by default; virtual-host style requires RUSTFS_SERVER_DOMAINS
		o.UsePathStyle = true

		if !secured {
			o.EndpointOptions.DisableHTTPS = true
		}
	})

	sgn := v4.NewSigner(func(so *v4.SignerOptions) {
		so.DisableURIPathEscaping = true // required, wrong signature otherwise
	})

	sgnwr := &uploadParamsFromContextPresigner{
		delegate: sgn,
	}

	presignClient := s3.NewPresignClient(s3client)

	return &S3Wrapper{
		Client:                           s3client,
		PresignClient:                    presignClient,
		uploadParamsFromContextPresigner: sgnwr,
		lgr:                              lgr,
	}, nil
}

type S3Wrapper struct {
	*s3.Client
	PresignClient                    *s3.PresignClient
	uploadParamsFromContextPresigner *uploadParamsFromContextPresigner
	lgr                              *logger.Logger
}

func (c *S3Wrapper) FileExists(ctx context.Context, bucket, key string) (bool, *s3.HeadObjectOutput, error) {
	objectInfo, err := c.Client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: &bucket,
		Key:    &key,
	})

	if err != nil {
		if isS3NotFoundError(err) {
			return false, nil, nil
		}
		return false, nil, err
	}
	return true, objectInfo, err
}

func isS3NotFoundError(err error) bool {
	if apiError, ok := errors.AsType[smithy.APIError](err); ok {
		switch apiError.(type) {
		case *types.NotFound:
			return true
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
	if isS3NotFoundError(err) {
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

func (c *S3Wrapper) ListObjectsV2WithMetadataAndTags(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*dto.ListObjectsV2Output, error) {
	// https://docs.aws.amazon.com/pdfs/sdk-for-go/v2/developer-guide/aws-sdk-go-v2-dg.pdf
	// Overriding Client Options For Operation Call
	ctx = dto.ContextWithRequestUserMetadata(ctx)
	ctx = dto.ContextWithCacheResponseToBytes(ctx)

	optFnsTo := []func(*s3.Options){}
	optFnsTo = append(optFnsTo, s3.WithAPIOptions(AddCacheBodyToSmthMiddleware, AddRequestUserMetadataMiddleware))
	optFnsTo = append(optFnsTo, optFns...)

	lst, err := c.Client.ListObjectsV2(ctx, params, optFnsTo...)
	if err != nil {
		return nil, fmt.Errorf("Error list objects with metadata & tags: %w", err)
	}

	rr, err := GetBytesCachedResponse(lst.ResultMetadata)
	if err != nil {
		return nil, fmt.Errorf("Unable to get cached response: %w", err)
	}

	listBucketResult := dto.ListObjectsV2Output{}
	if err = xmlDecoder(bytes.NewReader(rr), &listBucketResult); err != nil {
		return nil, err
	}

	listBucketResult.NextContinuationToken = lst.NextContinuationToken
	for i := range listBucketResult.Contents {
		mergeIn(&listBucketResult.Contents[i], &lst.Contents[i])
	}

	return &listBucketResult, nil
}

func mergeIn(into *dto.Object, src *types.Object) {
	into.ETag = src.ETag
	into.Key = src.Key
	into.LastModified = src.LastModified
	into.Owner = src.Owner
	into.Size = src.Size
}

func xmlDecoder(body io.Reader, v interface{}) error {
	d := xml.NewDecoder(body)
	return d.Decode(v)
}

func (c *S3Wrapper) PresignPutObjectProxyable(ctx context.Context, params *s3.PutObjectInput, uploadId, partNumber string, optFns ...func(*s3.PresignOptions)) (*v4.PresignedHTTPRequest, error) {
	// multipart upload proxying 1/6
	ctx = dto.ContextWithUpploadId(ctx, uploadId)
	ctx = dto.ContextWithPartNumber(ctx, partNumber)

	optFnsTo := []func(*s3.PresignOptions){}
	optFnsTo = append(optFnsTo, func(po *s3.PresignOptions) {
		po.Presigner = c.uploadParamsFromContextPresigner
	})
	optFnsTo = append(optFnsTo, optFns...)

	return c.PresignClient.PresignPutObject(ctx, params, optFnsTo...)
}
