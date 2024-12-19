package main

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	awsCredentials "github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	awsS3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/notification"
	"github.com/nkonev/dcron"
	log "github.com/sirupsen/logrus"
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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
	"nkonev.name/storage/client"
	"nkonev.name/storage/config"
	"nkonev.name/storage/handlers"
	"nkonev.name/storage/listener"
	. "nkonev.name/storage/logger"
	"nkonev.name/storage/producer"
	"nkonev.name/storage/rabbitmq"
	"nkonev.name/storage/s3"
	"nkonev.name/storage/services"
	"nkonev.name/storage/tasks"
	"nkonev.name/storage/utils"
)

const EXTERNAL_TRACE_ID_HEADER = "trace-id"
const TRACE_RESOURCE = "storage"

func main() {
	config.InitViper()
	lgr := NewLogger()

	app := fx.New(
		fx.Logger(lgr),
		fx.Supply(lgr),
		fx.Provide(
			configureTracer,
			configureInternalMinio,
			configureAwsS3,
			configureMinioEntities,
			configureEcho,
			tasks.RedisV9,
			tasks.RedisLocker,
			tasks.Scheduler,
			tasks.NewCleanFilesOfDeletedChatService,
			tasks.CleanFilesOfDeletedChatScheduler,
			tasks.NewActualizeGeneratedFilesService,
			tasks.ActualizeGeneratedFilesScheduler,
			client.NewChatAccessClient,
			handlers.ConfigureStaticMiddleware,
			handlers.ConfigureAuthMiddleware,
			handlers.NewUserAvatarHandler,
			handlers.NewChatAvatarHandler,
			handlers.NewFilesHandler,
			listener.CreateMinioEventsListener,
			producer.NewRabbitFileUploadedPublisher,
			rabbitmq.CreateRabbitMqConnection,
			services.NewFilesService,
			services.NewPreviewService,
			services.NewEventService,
			services.NewConvertingService,
			services.NewRedisInfoService,
		),
		fx.Invoke(
			runScheduler,
			runEcho,
			listener.CreateMinioEventsChannel,
		),
	)
	app.Run()

	lgr.Infof("Exit program")
}

func configureWriteHeaderMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) (err error) {
			handler := http.HandlerFunc(
				func(w http.ResponseWriter, r *http.Request) {
					ctx.SetRequest(r)
					ctx.SetResponse(echo.NewResponse(w, ctx.Echo()))
					existsSpan := trace.SpanFromContext(ctx.Request().Context())
					if existsSpan.SpanContext().HasTraceID() {
						w.Header().Set(EXTERNAL_TRACE_ID_HEADER, existsSpan.SpanContext().TraceID().String())
					}
					err = next(ctx)
				},
			)
			handler.ServeHTTP(ctx.Response(), ctx.Request())
			return
		}
	}
}

func configureOpentelemetryMiddleware(tp *sdktrace.TracerProvider) echo.MiddlewareFunc {
	mw := otelecho.Middleware(TRACE_RESOURCE, otelecho.WithTracerProvider(tp))
	return mw
}

func createCustomHTTPErrorHandler(lgr *log.Logger, e *echo.Echo) func(err error, c echo.Context) {
	originalHandler := e.DefaultHTTPErrorHandler
	return func(err error, c echo.Context) {
		GetLogEntry(c.Request().Context(), lgr).Errorf("Unhandled error: %v", err)
		originalHandler(err, c)
	}
}

func configureEcho(
	lgr *log.Logger,
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
	e.Logger.SetOutput(lgr.Writer())

	e.HTTPErrorHandler = createCustomHTTPErrorHandler(lgr, e)

	e.Pre(echo.MiddlewareFunc(staticMiddleware))
	e.Use(configureOpentelemetryMiddleware(tp))
	e.Use(configureWriteHeaderMiddleware())
	e.Use(echo.MiddlewareFunc(authMiddleware))
	accessLoggerConfig := middleware.LoggerConfig{
		Output: lgr.Writer(),
		Format: `"remote_ip":"${remote_ip}",` +
			`"method":"${method}","uri":"${uri}",` +
			`"status":${status},` +
			`,"bytes_in":${bytes_in},"bytes_out":${bytes_out},"traceId":"${header:uber-trace-id}"` + "\n",
	}
	e.Use(middleware.LoggerWithConfig(accessLoggerConfig))
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

func configureInternalMinio() (*s3.InternalMinioClient, error) {
	endpoint := viper.GetString("minio.internalEndpoint")
	accessKeyID := viper.GetString("minio.accessKeyId")
	secretAccessKey := viper.GetString("minio.secretAccessKey")
	location := viper.GetString("minio.location")
	secured := viper.GetBool("minio.secured")

	// Initialize minio client object.
	minioClient, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKeyID, secretAccessKey, ""),
		Secure: secured,
		Region: location,
	})
	if err != nil {
		return nil, err
	}

	return &s3.InternalMinioClient{minioClient}, nil
}

// https://github.com/aws/aws-sdk-go
func configureAwsS3() *awsS3.S3 {
	endpoint := viper.GetString("minio.internalEndpoint")
	accessKeyID := viper.GetString("minio.accessKeyId")
	secretAccessKey := viper.GetString("minio.secretAccessKey")
	location := viper.GetString("minio.location")
	secured := viper.GetBool("minio.secured")

	creds := awsCredentials.NewStaticCredentials(accessKeyID, secretAccessKey, "")

	nonSecured := !secured

	forcePath := true
	cfg := aws.Config{
		Endpoint:         &endpoint,
		Credentials:      creds,
		S3ForcePathStyle: &forcePath,
		Region:           &location,
		DisableSSL:       &nonSecured,
	}
	sess := session.Must(session.NewSession(&cfg))
	svc := awsS3.New(sess)
	return svc
}

func configureTracer(lgr *log.Logger, lc fx.Lifecycle) (*sdktrace.TracerProvider, error) {
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
				lgr.Printf("Error shutting down tracer provider: %v", err)
			}
			return nil
		},
	})

	return tp, nil
}

// rely on viper import and it's configured by
func runEcho(lgr *log.Logger, e *echo.Echo) {
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

func configureMinioEntities(lgr *log.Logger, client *s3.InternalMinioClient) (*utils.MinioConfig, error) {
	var ua, ca, f, p string
	var err error
	if ua, err = utils.EnsureAndGetUserAvatarBucket(lgr, client); err != nil {
		return nil, err
	}
	if ca, err = utils.EnsureAndGetChatAvatarBucket(lgr, client); err != nil {
		return nil, err
	}
	if f, err = utils.EnsureAndGetFilesBucket(lgr, client); err != nil {
		return nil, err
	}
	if p, err = utils.EnsureAndGetFilesPreviewBucket(lgr, client); err != nil {
		return nil, err
	}
	bucketNotification, err := client.GetBucketNotification(context.Background(), f)
	if err != nil {
		return nil, err
	}

	arn := notification.Arn{
		Partition: "minio",
		Service:   "sqs",
		Region:    "",
		AccountID: "primary",
		Resource:  "amqp",
	}
	subscriptionName := arn.String()
	shouldCreateSubscription := true
	queueConfigs := bucketNotification.QueueConfigs
	if queueConfigs != nil {
		for _, qc := range queueConfigs {
			if qc.Queue == subscriptionName {
				shouldCreateSubscription = false
				break
			}
		}
	}
	if shouldCreateSubscription {
		lgr.Infof("Will create subscription for bucket %v to arn %v", f, arn)
		err := client.SetBucketNotification(context.Background(), f, notification.Configuration{
			QueueConfigs: []notification.QueueConfig{
				notification.QueueConfig{
					Queue: subscriptionName,
					Config: notification.Config{
						Events: []notification.EventType{
							utils.ObjectCreated + ":*",
							utils.ObjectRemoved + ":*",
						},
					},
				},
			},
		})
		if err != nil {
			return nil, err
		}
	}
	return &utils.MinioConfig{
		UserAvatar:   ua,
		ChatAvatar:   ca,
		Files:        f,
		FilesPreview: p,
	}, nil
}

func runScheduler(
	lgr *log.Logger,
	scheduler *dcron.Cron,
	dt *tasks.CleanFilesOfDeletedChatTask,
	a *tasks.ActualizeGeneratedFilesTask,
	lc fx.Lifecycle,
) error {
	scheduler.Start()
	lgr.Infof("Scheduler started")

	if viper.GetBool("schedulers." + dt.Key() + ".enabled") {
		lgr.Infof("Adding " + dt.Key() + " job to scheduler")
		err := scheduler.AddJobs(dt)
		if err != nil {
			return err
		}
	}

	if viper.GetBool("schedulers." + a.Key() + ".enabled") {
		lgr.Infof("Adding " + a.Key() + " job to scheduler")
		err := scheduler.AddJobs(a)
		if err != nil {
			return err
		}
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
