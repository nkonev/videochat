package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"nkonev.name/chat/app"
	"nkonev.name/chat/config"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.uber.org/fx"
)

const headerTraceId = "X-Traceid"
const headerCorrelationId = "X-CorrelationId"

const gitJson = "git.json"

func CreateHttpRouter(
	cfg *config.AppConfig,
	lgr *logger.LoggerWrapper,
	chatHandler *ChatHandler,
	participantHandler *ParticipantHandler,
	messageHandler *MessageHandler,
	blogHandler *BlogHandler,
	technicalHandler *TechnicalHandler,
	staticHandler *StaticHandler,
) *gin.Engine {
	// https://gin-gonic.com/en/docs/examples/graceful-restart-or-stop/
	gin.SetMode(gin.ReleaseMode)
	ginRouter := gin.New()
	ginRouter.Use(otelgin.Middleware(app.TRACE_RESOURCE))
	ginRouter.Use(StructuredLogMiddleware(lgr))
	ginRouter.Use(WriteTraceToHeaderMiddleware())
	if cfg.Server.Dump {
		ginRouter.Use(DumpMiddleware(lgr, cfg))
	}
	ginRouter.Use(gin.Recovery())

	ginRouter.POST("/api/chat", chatHandler.CreateChat)
	ginRouter.PUT("/api/chat/tet-a-tet/:participantId", chatHandler.CreateTetAChat)
	ginRouter.PUT("/api/chat", chatHandler.EditChat)
	ginRouter.DELETE("/api/chat/:id", chatHandler.DeleteChat)
	ginRouter.PUT("/api/chat/:id/pin", chatHandler.PinChat)
	ginRouter.GET("/api/chat/:id", chatHandler.GetChat)
	ginRouter.GET("/api/chat/search", chatHandler.SearchChats)
	ginRouter.POST("/api/chat/fresh", chatHandler.ChatsFresh)
	ginRouter.POST("/api/chat/filter", chatHandler.ChatsFilter)

	ginRouter.PUT("/api/chat/:id/notification", chatHandler.PutUserChatNotificationSettings)
	ginRouter.GET("/api/chat/:id/notification", chatHandler.GetUserChatNotificationSettings)
	ginRouter.GET("/api/chat/has-new-messages", chatHandler.HasNewMessages)

	ginRouter.PUT("/api/chat/:id/participant", participantHandler.AddParticipant)
	ginRouter.DELETE("/api/chat/:id/participant/:participantId", participantHandler.DeleteParticipant)
	ginRouter.GET("/api/chat/:id/participant/search", participantHandler.SearchParticipants)
	ginRouter.PUT("/api/chat/:id/participant/:participantId", participantHandler.ChangeParticipant)
	ginRouter.POST("/api/chat/:id/participant/filter", participantHandler.ParticipantsFilter)
	ginRouter.GET("/api/chat/:id/user-candidate", participantHandler.SearchForUsersToAdd)
	ginRouter.POST("/api/chat/:id/participant/count", participantHandler.CountParticipants)
	ginRouter.PUT("/api/chat/:id/leave", participantHandler.LeaveChat)
	ginRouter.PUT("/api/chat/:id/join", participantHandler.JoinChat)

	ginRouter.POST("/api/chat/:id/message", messageHandler.CreateMessage)
	ginRouter.PUT("/api/chat/:id/message", messageHandler.EditMessage)
	ginRouter.PUT("/api/chat/:id/message/:messageId/sync-embed", messageHandler.SyncEmbed)
	ginRouter.DELETE("/api/chat/:id/message/:messageId", messageHandler.DeleteMessage)
	ginRouter.GET("/api/chat/:id/message/read/:messageId", messageHandler.GetReadMessageUsers)
	ginRouter.PUT("/api/chat/:id/message/read/:messageId", messageHandler.ReadMessage)
	ginRouter.PUT("/api/chat/:id/read", messageHandler.MarkChatAsRead)
	ginRouter.PUT("/api/chat/read", messageHandler.MarkAsReadAllChats)
	ginRouter.GET("/api/chat/:id/message/search", messageHandler.SearchMessages)
	ginRouter.PUT("/api/chat/:id/message/:messageId/blog-post", messageHandler.MakeBlogPost)
	ginRouter.PUT("/api/chat/:id/message/:messageId/reaction", messageHandler.ReactionMessage)
	ginRouter.POST("/api/chat/:id/message/fresh", messageHandler.MessagesFresh)
	ginRouter.POST("/api/chat/:id/message/filter", messageHandler.MessagesFilter)
	ginRouter.PUT("/api/chat/public/preview-without-html", messageHandler.MessagePreview)
	ginRouter.GET("/api/chat/:id/mention/suggest", messageHandler.SearchForUsersToMention)
	ginRouter.GET("/api/chat/:id/message/find-by-file-item-uuid/:fileItemUuid", messageHandler.FindMessageByFileItemUuid)
	ginRouter.PUT("/api/chat/:id/message/file-item-uuid", messageHandler.SetFileItemUuid)
	ginRouter.GET("/api/chat/:id/message/pin", messageHandler.GetPinnedMessages)
	ginRouter.GET("/api/chat/:id/message/pin/promoted", messageHandler.GetPinnedPromotedMessage)
	ginRouter.PUT("/api/chat/:id/message/:messageId/pin", messageHandler.PinMessage)
	ginRouter.PUT("/api/chat/:id/message/:messageId/publish", messageHandler.PublishMessage)
	ginRouter.GET("/api/chat/:id/message/publish", messageHandler.GetPublishedMessages)
	ginRouter.GET("/api/chat/public/:id/message/:messageId", messageHandler.GetPublishedMessageForPublic)

	ginRouter.PUT("/api/chat/:id/typing", messageHandler.TypeMessage)
	ginRouter.PUT("/api/chat/:id/broadcast", messageHandler.BroadcastMessage)

	ginRouter.GET("/api/blog", blogHandler.SearchBlogs)
	ginRouter.GET("/api/blog/:id", blogHandler.GetBlog)
	ginRouter.GET("/api/blog/:id/comment", blogHandler.SearchComments)
	ginRouter.GET("/api/chat/can-create-blog", blogHandler.CanCreateBlog)
	ginRouter.GET("/internal/blog/seo", blogHandler.GetAllBlogPostsForSeo)

	ginRouter.GET("/internal/access", chatHandler.CheckAccess)
	ginRouter.GET("/internal/is-admin", chatHandler.IsAdmin)
	ginRouter.GET("/internal/does-chats-exist", chatHandler.IsExists)
	ginRouter.GET("/internal/participant-ids", participantHandler.GetChatParticipants)
	ginRouter.GET("/internal/basic/:id", chatHandler.GetBasicInfo)
	ginRouter.GET("/internal/name-for-invite", chatHandler.GetNameForInvite)

	ginRouter.GET("/internal/health", technicalHandler.Health)
	if cfg.Cqrs.TestHelperMethods {
		ginRouter.DELETE("/internal/truncate", technicalHandler.Truncate)
	}

	ginRouter.GET("/"+gitJson, staticHandler.StaticGitJson)

	return ginRouter
}

func getUserId(g *gin.Context) (int64, error) {
	uh := g.Request.Header.Get(utils.HeaderUserId)
	return utils.ParseInt64(uh)
}

func getUserLogin(g *gin.Context) (string, error) {
	decodedStringBytes, err := base64.StdEncoding.DecodeString(g.Request.Header.Get(utils.HeaderUserLogin))
	if err != nil {
		return "", err
	}

	return string(decodedStringBytes), nil
}

func getUserRoles(g *gin.Context) []string {
	return g.Request.Header.Values(utils.HeaderUserRole)
}

func getUserPermissions(g *gin.Context) []string {
	return g.Request.Header.Values(utils.HeaderUserPermission)
}

func getCorrelationId(g *gin.Context) *string {
	ch := g.Request.Header.Get(headerCorrelationId)
	if len(ch) > 0 {
		_, err := uuid.Parse(ch)
		if err == nil {
			return &ch
		}
	}
	return nil
}

func getQueryParamsAsInt64Slice(g *gin.Context, queryParamName string) ([]int64, error) {
	uids := g.Query(queryParamName)
	uidss := strings.Split(uids, ",")
	var ids []int64 = make([]int64, 0)
	for _, iu := range uidss {
		pa, err := utils.ParseInt64(iu)
		if err != nil {
			return nil, err
		}
		ids = append(ids, pa)
	}
	return ids, nil
}

func ConfigureHttpServer(
	cfg *config.AppConfig,
	lgr *logger.LoggerWrapper,
	lc fx.Lifecycle,
	ginRouter *gin.Engine,
) *http.Server {
	httpServer := &http.Server{
		Addr:           cfg.Server.Address,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		MaxHeaderBytes: cfg.Server.MaxHeaderBytes,
		Handler:        ginRouter.Handler(),
	}

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			lgr.Info("Stopping http server")

			if err := httpServer.Shutdown(context.Background()); err != nil {
				lgr.Error("Error shutting http server", logger.AttributeError, err)
			}
			return nil
		},
	})

	return httpServer
}

type ResponseWriterWrapper struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	statusCode *int
}

// NewResponseWriterWrapper static function creates a wrapper for the http.ResponseWriter
func NewResponseWriterWrapper(w gin.ResponseWriter) ResponseWriterWrapper {
	var buf bytes.Buffer
	var statusCode int = 200
	return ResponseWriterWrapper{
		ResponseWriter: w,
		body:           &buf,
		statusCode:     &statusCode,
	}
}

func (rww ResponseWriterWrapper) Write(buf []byte) (int, error) {
	rww.body.Write(buf)
	return rww.ResponseWriter.Write(buf)
}

// Header function overwrites the http.ResponseWriter Header() function
func (rww ResponseWriterWrapper) Header() http.Header {
	return rww.ResponseWriter.Header()
}

// WriteHeader function overwrites the http.ResponseWriter WriteHeader() function
func (rww ResponseWriterWrapper) WriteHeader(statusCode int) {
	(*rww.statusCode) = statusCode
	rww.ResponseWriter.WriteHeader(statusCode)
}

func (rww ResponseWriterWrapper) String() string {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("HTTP %d\n", *(rww.statusCode)))

	for k, v := range rww.ResponseWriter.Header() {
		buf.WriteString(fmt.Sprintf("%s: %v\n", k, v))
	}
	buf.WriteString("\n")

	buf.WriteString(rww.body.String())
	buf.WriteString("\n")

	return buf.String()
}

func StructuredLogMiddleware(lgr *logger.LoggerWrapper) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()

		// Start timer
		start := time.Now()

		// Process Request
		c.Next()

		// Stop timer
		end := time.Now()

		duration := end.Sub(start)

		entries := []any{
			"client_ip", c.ClientIP(),
			"duration", duration,
			"method", c.Request.Method,
			"path", c.Request.RequestURI,
			"status", c.Writer.Status(),
			"referrer", c.Request.Referer(),
		}

		if c.Writer.Status() >= 500 {
			lgr.ErrorContext(ctx, "Request", entries...)
		} else {
			lgr.InfoContext(ctx, "Request", entries...)
		}
	}
}

func WriteTraceToHeaderMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceId := logger.GetTraceId(c.Request.Context())

		c.Writer.Header().Set(headerTraceId, traceId)

		// Process Request
		c.Next()

	}
}

func DumpMiddleware(lgr *logger.LoggerWrapper, cfg *config.AppConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// https://stackoverflow.com/questions/66528234/log-http-responsewriter-content
		rww := NewResponseWriterWrapper(c.Writer)
		// w.Header()
		c.Writer = rww

		dumpReq, err := httputil.DumpRequest(c.Request, true)
		if err != nil {
			lgr.ErrorContext(c.Request.Context(), "Error during dumping http request", logger.AttributeError, err)
		} else {
			if cfg.Server.PrettyLog && !cfg.Logger.Json {
				fmt.Printf(">>> HTTP REQUEST trace_id=%s\n", logger.GetTraceId(c.Request.Context()))
				fmt.Printf("%s\n", string(dumpReq))
			} else {
				lgr.InfoContext(c.Request.Context(), fmt.Sprintf(">>> HTTP REQUEST %s", string(dumpReq)))
			}
		}

		c.Next()

		if cfg.Server.PrettyLog && !cfg.Logger.Json {
			fmt.Printf("<<< HTTP RESPONSE trace_id=%s \n%s\n", logger.GetTraceId(c.Request.Context()), rww.String())
		} else {
			lgr.InfoContext(c.Request.Context(), "<<< HTTP RESPONSE "+rww.String())
		}
	}
}

func RunHttpServer(
	lgr *logger.LoggerWrapper,
	httpServer *http.Server,
	cfg *config.AppConfig,
) {
	go func() {
		lgr.InfoContext(context.Background(), "http server is configured with address", "http_address", cfg.Server.Address)
		err := httpServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			lgr.Info("Http server is closed")
		} else if err != nil {
			lgr.Error("Got http server error", logger.AttributeError, err)
			panic(err)
		}
	}()
}
