package main

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/minio/minio-go/v7"
	minioCredentials "github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/nkonev/dcron"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/github.com/labstack/echo/otelecho"
	jaegerPropagator "go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.10.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"nkonev.name/storage/app"
	"nkonev.name/storage/client"
	"nkonev.name/storage/config"
	"nkonev.name/storage/db"
	"nkonev.name/storage/dto"
	"nkonev.name/storage/handlers"
	"nkonev.name/storage/listener"
	"nkonev.name/storage/logger"
	"nkonev.name/storage/producer"
	"nkonev.name/storage/rabbitmq"
	"nkonev.name/storage/services"
	"nkonev.name/storage/tasks"
	"nkonev.name/storage/utils"
)

const EXTERNAL_TRACE_ID_HEADER = "X-Traceid"
const TRACE_RESOURCE = app.APP_NAME

func main() {
	config.InitViper()
	lgr := logger.NewLogger()
	defer lgr.CloseLogger()

	appFx := fx.New(
		fx.Supply(lgr),
		fx.WithLogger(func(log *logger.Logger) fxevent.Logger {
			return &fxevent.ZapLogger{Logger: log.ZapLogger}
		}),
		fx.Provide(
			configureTracer,
			configureInternalMinio,
			configureAwsS3v2,
			configureMinioEntities,
			configureEcho,
			db.ConfigureDb,
			configureMigrations,
			tasks.RedisV9,
			tasks.Scheduler,
			tasks.NewCleanFilesOfDeletedChatService,
			tasks.CleanFilesOfDeletedChatScheduler,
			tasks.NewActualizeGeneratedFilesService,
			tasks.ActualizeGeneratedFilesScheduler,
			tasks.NewActualizeMetadataCacheService,
			tasks.ActualizeMetadataCacheScheduler,
			client.NewChatAccessClient,
			handlers.ConfigureStaticMiddleware,
			handlers.ConfigureAuthMiddleware,
			handlers.NewUserAvatarHandler,
			handlers.NewChatAvatarHandler,
			handlers.NewFilesHandler,
			listener.CreateS3EventListener,
			producer.NewRabbitFileUploadedPublisher,
			rabbitmq.CreateRabbitMqConnection,
			services.NewFilesService,
			services.NewPreviewService,
			services.NewEventService,
			services.NewConvertingService,
			services.NewRedisInfoService,
		),
		fx.Invoke(
			runMigrations,
			runScheduler,
			runEcho,
			listener.CreateMinioEventsChannel,
		),
	)
	appFx.Run()

	lgr.Infof("Exit program")
}

func configureWriteHeaderMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			existsSpan := trace.SpanFromContext(c.Request().Context())
			if existsSpan.SpanContext().HasTraceID() {
				c.Response().Header().Set(EXTERNAL_TRACE_ID_HEADER, existsSpan.SpanContext().TraceID().String())
			}
			if err := next(c); err != nil {
				c.Error(err)
			}
			return nil
		}
	}
}

func configureOpentelemetryMiddleware(tp *sdktrace.TracerProvider) echo.MiddlewareFunc {
	mw := otelecho.Middleware(TRACE_RESOURCE, otelecho.WithTracerProvider(tp))
	return mw
}

func createCustomHTTPErrorHandler(lgr *logger.Logger, e *echo.Echo) func(err error, c echo.Context) {
	originalHandler := e.DefaultHTTPErrorHandler
	return func(err error, c echo.Context) {
		if c.Response().Committed {
			return
		}

		lgr.WithTracing(c.Request().Context()).Errorf("Unhandled error: %v", err)
		originalHandler(err, c)
	}
}

func configureEcho(
	lgr *logger.Logger,
	staticMiddleware handlers.StaticMiddleware,
	authMiddleware handlers.AuthMiddleware,
	lc fx.Lifecycle,
	uah *handlers.UserAvatarHandler,
	cha *handlers.ChatAvatarHandler,
	fh *handlers.FilesHandler,
	tp *sdktrace.TracerProvider,
) *echo.Echo {

	bodyLimit := viper.GetString("server.body.limit")

	e := echo.New()
	e.Logger.SetOutput(lgr)

	e.HTTPErrorHandler = createCustomHTTPErrorHandler(lgr, e)

	e.Pre(echo.MiddlewareFunc(staticMiddleware))
	e.Use(configureOpentelemetryMiddleware(tp))
	skipper := func(c echo.Context) bool {
		// Skip health check endpoint
		return c.Request().URL.Path == "/health"
	}
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:        true,
		LogURI:           true,
		LogMethod:        true,
		LogRemoteIP:      true,
		LogError:         true,
		LogLatency:       true,
		LogUserAgent:     true,
		LogContentLength: true,
		LogResponseSize:  true,
		Skipper:          skipper,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			tl := lgr.SugaredLogger
			spanCtx := trace.SpanContextFromContext(c.Request().Context())
			if spanCtx.HasTraceID() {
				tl = lgr.With(
					zap.String("trace_id", spanCtx.TraceID().String()),
					zap.String("span_id", spanCtx.SpanID().String()),
				)
			}
			tl = tl.With(
				"status", v.Status,
				"uri", v.URI,
				"method", v.Method,
				"remote_ip", v.RemoteIP,
				"latency", v.Latency,
				"user_agent", v.UserAgent,
				"content_length", v.ContentLength,
				"response_size", v.ResponseSize,
			)

			if v.Error == nil {
				tl.Infof("REQUEST")
			} else {
				tl = tl.With(
					"error", v.Error.Error(),
				)
				tl.Errorf("REQUEST")
			}
			return nil
		},
	}))
	e.Use(configureWriteHeaderMiddleware())
	e.Use(echo.MiddlewareFunc(authMiddleware))
	e.Use(middleware.Secure())
	e.Use(middleware.BodyLimit(bodyLimit))

	e.POST("/api/storage/avatar", uah.PutAvatar)
	e.GET(fmt.Sprintf("%v/:filename", uah.GetUrlPath()), uah.Download)
	e.POST("/api/storage/chat/:chatId/avatar", cha.PutAvatar)
	e.GET(fmt.Sprintf("%v/:filename", cha.GetUrlPath()), cha.Download)
	e.POST("/internal/s3", fh.S3Handler)
	e.PUT("/api/storage/:chatId/upload/init", fh.InitMultipartUpload)
	e.PUT("/api/storage/:chatId/upload/finish", fh.FinishMultipartUpload)
	e.PUT("/api/storage/:chatId/replace/file", fh.ReplaceHandler)
	e.GET("/api/storage/:chatId", fh.ListHandler)
	e.GET("/api/storage/public/:chatId", fh.ListHandlerPublic)
	e.POST("/api/storage/view/list", fh.ViewListHandler)
	e.POST("/api/storage/public/view/list", fh.ViewListHandler)
	e.POST("/api/storage/view/status", fh.ViewStatusHandler)
	e.POST("/api/storage/public/view/status", fh.ViewStatusHandler)
	e.DELETE("/api/storage/:chatId/file", fh.DeleteHandler)
	e.PUT("/api/storage/publish/file", fh.SetPublic)
	e.POST("/api/storage/:chatId/file/count", fh.CountHandler)
	e.POST("/api/storage/:chatId/file/filter", fh.FilterHandler)
	e.GET("/api/storage/:chatId/file-item-uuid", fh.ListFileItemUuids)
	e.GET("/api/storage/:chatId/file", fh.LimitsHandler)
	e.GET("/api/storage/:chatId/embed/candidates", fh.ListCandidatesForEmbed)
	e.POST("/api/storage/:chatId/embed/filter", fh.FilterEmbed)
	e.POST("/api/storage/:chatId/embed/count", fh.CountEmbed)
	e.GET("/api/storage/embed/preview", fh.PreviewDownloadHandler)
	e.GET(utils.UrlStoragePublicPreviewFile, fh.PublicPreviewDownloadHandler)
	e.GET(utils.UrlStoragePublicGetFile, fh.PublicDownloadHandler)
	e.GET(utils.UrlStorageGetFile, fh.DownloadHandler)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			// do some work on application stop (like closing connections and files)
			lgr.Infof("Stopping http server")
			return e.Shutdown(ctx)
		},
	})

	return e
}

func configureInternalMinio() (*client.InternalMinioClient, error) {
	endpoint := viper.GetString("minio.internalEndpointMinio")
	accessKeyID := viper.GetString("minio.accessKeyId")
	secretAccessKey := viper.GetString("minio.secretAccessKey")
	location := viper.GetString("minio.location")
	secured := viper.GetBool("minio.secured")

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  minioCredentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: secured,
		Region: location,
	})
	if err != nil {
		return nil, err
	}

	return &client.InternalMinioClient{minioClient}, nil
}

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

// https://docs.rustfs.com/en/developer/sdk/go
// https://docs.aws.amazon.com/code-library/latest/ug/go_2_s3_code_examples.html
func configureAwsS3v2(lgr *logger.Logger) (*client.S3Wrapper, error) {
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
		so.DisableURIPathEscaping = true
	})

	sgnwr := &uploadParamsFromContextPresigner{
		delegate: sgn,
	}

	presignClient := s3.NewPresignClient(s3client, func(po *s3.PresignOptions) {
		po.Presigner = sgnwr
	})

	return client.NewS3Wrapper(s3client, presignClient, lgr), nil
}

func configureTracer(lgr *logger.Logger, lc fx.Lifecycle) (*sdktrace.TracerProvider, error) {
	lgr.Infof("Configuring Jaeger tracing")
	conn, err := grpc.DialContext(context.Background(), viper.GetString("otlp.endpoint"), grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, err
	}

	exporter, err := otlptracegrpc.New(context.Background(), otlptracegrpc.WithGRPCConn(conn))
	if err != nil {
		return nil, err
	}
	resources := resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.ServiceNameKey.String(TRACE_RESOURCE),
	)
	batchSpanProcessor := sdktrace.NewBatchSpanProcessor(exporter)
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithSpanProcessor(batchSpanProcessor),
		sdktrace.WithResource(resources),
	)
	otel.SetTracerProvider(tp)
	aJaegerPropagator := jaegerPropagator.Jaeger{}
	// register jaeger propagator
	otel.SetTextMapPropagator(aJaegerPropagator)
	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			lgr.Infof("Stopping tracer")
			if err := tp.Shutdown(context.Background()); err != nil {
				lgr.Errorf("Error shutting down tracer provider: %v", err)
			}
			return nil
		},
	})

	return tp, nil
}

func configureMigrations() *db.MigrationsConfig {
	return &db.MigrationsConfig{}
}

func runMigrations(lgr *logger.Logger, db *db.DB, migrationsConfig *db.MigrationsConfig) {
	db.Migrate(lgr, migrationsConfig)
}

// rely on viper import and it's configured by
func runEcho(lgr *logger.Logger, e *echo.Echo) {
	address := viper.GetString("server.address")

	lgr.Info("Starting server...")
	// Start server in another goroutine
	go func() {
		if err := e.Start(address); err != nil {
			lgr.Infof("server shut down: %v", err)
		}
	}()
	lgr.Info("Server started. Waiting for interrupt signal 2 (Ctrl+C)")
}

func ensureAndGetUserAvatarBucket(ctx context.Context, s3Client *client.S3Wrapper) (string, error) {
	bucketName := viper.GetString("minio.bucket.userAvatar")
	err := s3Client.EnsureBucket(ctx, bucketName)
	return bucketName, err
}

func ensureAndGetChatAvatarBucket(ctx context.Context, s3Client *client.S3Wrapper) (string, error) {
	bucketName := viper.GetString("minio.bucket.chatAvatar")
	err := s3Client.EnsureBucket(ctx, bucketName)
	return bucketName, err
}

func ensureAndGetFilesBucket(ctx context.Context, s3Client *client.S3Wrapper) (string, error) {
	bucketName := viper.GetString("minio.bucket.files")
	err := s3Client.EnsureBucket(ctx, bucketName)
	return bucketName, err
}

func ensureAndGetFilesPreviewBucket(ctx context.Context, s3Client *client.S3Wrapper) (string, error) {
	bucketName := viper.GetString("minio.bucket.filesPreview")
	err := s3Client.EnsureBucket(ctx, bucketName)
	return bucketName, err
}

func configureMinioEntities(lgr *logger.Logger, s3Client *client.S3Wrapper) (*utils.MinioConfig, error) {
	var ua, ca, f, p string
	var err error

	ctx := context.Background()
	if ua, err = ensureAndGetUserAvatarBucket(ctx, s3Client); err != nil {
		return nil, err
	}
	if ca, err = ensureAndGetChatAvatarBucket(ctx, s3Client); err != nil {
		return nil, err
	}
	if f, err = ensureAndGetFilesBucket(ctx, s3Client); err != nil {
		return nil, err
	}
	if p, err = ensureAndGetFilesPreviewBucket(ctx, s3Client); err != nil {
		return nil, err
	}

	bucketNotification, err := s3Client.GetBucketNotificationConfiguration(ctx, &s3.GetBucketNotificationConfigurationInput{
		Bucket: &f,
	})
	if err != nil {
		return nil, err
	}

	location := viper.GetString("minio.location")

	arn := "arn:rustfs:sqs:" + location + ":primary:amqp"

	shouldCreateSubscription := true
	queueConfigs := bucketNotification.QueueConfigurations
	for _, qc := range queueConfigs {
		if *qc.QueueArn == arn {
			shouldCreateSubscription = false
			break
		}
	}
	if shouldCreateSubscription {
		lgr.Infof("Will create subscription for bucket %v to arn %v", f, arn)

		bucketName := f

		output, err := s3Client.PutBucketNotificationConfiguration(ctx, &s3.PutBucketNotificationConfigurationInput{
			Bucket: new(bucketName),
			NotificationConfiguration: &types.NotificationConfiguration{
				QueueConfigurations: []types.QueueConfiguration{
					types.QueueConfiguration{
						QueueArn: &arn,
						Events: []types.Event{
							types.Event(dto.ObjectCreated + ":*"),
							types.Event(dto.ObjectRemoved + ":*"),
						},
					},
				},
			},
		})
		if err != nil {
			return nil, err
		}
		lgr.Info("PutBucketNotificationConfiguration response is: %v", output)
	}
	return &utils.MinioConfig{
		UserAvatar:   ua,
		ChatAvatar:   ca,
		Files:        f,
		FilesPreview: p,
	}, nil
}

func runScheduler(
	lgr *logger.Logger,
	scheduler *dcron.Cron,
	dt *tasks.CleanFilesOfDeletedChatTask,
	a *tasks.ActualizeGeneratedFilesTask,
	ma *tasks.ActualizeMetadataCacheTask,
	lc fx.Lifecycle,
) error {
	scheduler.Start()
	lgr.Infof("Scheduler started")

	if viper.GetBool("schedulers." + dt.Key() + ".enabled") {
		lgr.Infof("Adding task " + dt.Key() + " to scheduler")
		err := scheduler.AddJobs(dt)
		if err != nil {
			return err
		}
	} else {
		lgr.Infof("Task " + dt.Key() + " is disabled")
	}

	if viper.GetBool("schedulers." + a.Key() + ".enabled") {
		lgr.Infof("Adding task " + a.Key() + " to scheduler")
		err := scheduler.AddJobs(a)
		if err != nil {
			return err
		}
	} else {
		lgr.Infof("Task " + a.Key() + " is disabled")
	}

	if viper.GetBool("schedulers." + ma.Key() + ".enabled") {
		lgr.Infof("Adding task " + ma.Key() + " to scheduler")
		err := scheduler.AddJobs(ma)
		if err != nil {
			return err
		}
	} else {
		lgr.Infof("Task " + ma.Key() + " is disabled")
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			lgr.Infof("Stopping scheduler")
			<-scheduler.Stop().Done()
			return nil
		},
	})
	return nil
}
