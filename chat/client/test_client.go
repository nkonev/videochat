package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"nkonev.name/chat/config"
	"nkonev.name/chat/dto"
	"nkonev.name/chat/logger"
	"nkonev.name/chat/utils"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
)

type TestRestClient struct {
	restClient
}

func NewTestRestClient(cfg *config.AppConfig, lgr *logger.LoggerWrapper) *TestRestClient {
	tr := &http.Transport{
		MaxIdleConns:       cfg.Http.MaxIdleConns,
		IdleConnTimeout:    cfg.Http.IdleConnTimeout,
		DisableCompression: cfg.Http.DisableCompression,
	}
	tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	trR := otelhttp.NewTransport(tr)
	client := &http.Client{Transport: trR}
	trcr := otel.Tracer("test/rest/client")

	return &TestRestClient{restClient{client, "http://localhost" + cfg.Server.Address, trcr, cfg, lgr, "[test http client]"}}
}

type ChatCreateOption interface {
	Apply(createDto *dto.ChatBaseCreateDto)
}

type ChatParamResend struct {
	v bool
}

func NewChatOptionResend(v bool) *ChatParamResend {
	return &ChatParamResend{v: v}
}

func (r *ChatParamResend) Apply(d *dto.ChatBaseCreateDto) {
	d.CanResend = &r.v
}

type ChatParamBlog struct {
	blog bool
}

func NewChatOptionBlog(blog bool) *ChatParamBlog {
	return &ChatParamBlog{blog: blog}
}

func (r *ChatParamBlog) Apply(d *dto.ChatBaseCreateDto) {
	d.Blog = r.blog
}

type ChatParamAvatar struct {
	avatar    *string
	avatarBig *string
}

func NewChatOptionAvatar(avatar, avatarBig *string) *ChatParamAvatar {
	return &ChatParamAvatar{avatar: avatar, avatarBig: avatarBig}
}

func (r *ChatParamAvatar) Apply(d *dto.ChatBaseCreateDto) {
	d.Avatar = r.avatar
	d.AvatarBig = r.avatarBig
}

type ChatParamParticipants struct {
	participants []int64
}

func NewChatOptionParticipants(participants ...int64) *ChatParamParticipants {
	return &ChatParamParticipants{participants: participants}
}

func (r *ChatParamParticipants) Apply(d *dto.ChatBaseCreateDto) {
	d.ParticipantIds = r.participants
}

func (rc *TestRestClient) CreateChat(ctx context.Context, behalfUserId int64, chatName string, chatCreateOptions ...ChatCreateOption) (int64, error) {
	ccd := dto.ChatBaseCreateDto{
		Title: chatName,
	}

	for _, opt := range chatCreateOptions {
		if opt != nil {
			opt.Apply(&ccd)
		}
	}

	req := dto.ChatCreateDto{
		ChatBaseCreateDto: ccd,
	}

	resp, err := query[dto.ChatCreateDto, dto.IdResponse](ctx, &rc.restClient, behalfUserId, http.MethodPost, "/api/chat", "chat.Create", &req, nil)
	if err != nil {
		return 0, err
	}
	return resp.Id, nil
}

func (rc *TestRestClient) CreateTetATetChat(ctx context.Context, behalfUserId int64, oppositeUserId int64) (int64, error) {
	strUrl := fmt.Sprintf("/api/chat/tet-a-tet/%d", oppositeUserId)

	resp, err := query[any, dto.IdResponse](ctx, &rc.restClient, behalfUserId, http.MethodPut, strUrl, "chat.CreateTetATet", nil, nil)
	if err != nil {
		return 0, err
	}
	return resp.Id, nil
}

func (rc *TestRestClient) EditChat(ctx context.Context, behalfUserId int64, chatId int64, chatName string, chatCreateOptions ...ChatCreateOption) error {
	ccd := dto.ChatBaseCreateDto{
		Title: chatName,
	}

	for _, opt := range chatCreateOptions {
		if opt != nil {
			opt.Apply(&ccd)
		}
	}

	req := dto.ChatEditDto{
		Id:                chatId,
		ChatBaseCreateDto: ccd,
	}
	err := queryNoResponse[dto.ChatEditDto](ctx, &rc.restClient, behalfUserId, http.MethodPut, "/api/chat", "chat.Edit", &req, nil)
	if err != nil {
		return err
	}
	return nil
}

func (rc *TestRestClient) PinChat(ctx context.Context, behalfUserId int64, chatId int64, pin bool) error {
	return queryNoResponse[any](ctx, &rc.restClient, behalfUserId, http.MethodPut, "/api/chat/"+utils.ToString(chatId)+"/pin?pin="+utils.ToString(pin), "chat.Pin", nil, nil)
}

func (rc *TestRestClient) DeleteChat(ctx context.Context, behalfUserId int64, chatId int64) error {
	return queryNoResponse[any](ctx, &rc.restClient, behalfUserId, http.MethodDelete, "/api/chat/"+utils.ToString(chatId), "chat.Delete", nil, nil)
}

type ChatGetOption interface {
	Apply(queryParams *url.Values) *url.Values
}

type ChatGetOptionWithSize struct {
	v int32
}

func NewChatGetOptionWithSize(v int32) *ChatGetOptionWithSize {
	return &ChatGetOptionWithSize{v: v}
}

func (r *ChatGetOptionWithSize) Apply(queryParams *url.Values) *url.Values {
	if queryParams == nil {
		queryParams = &url.Values{}
	}
	queryParams.Add(dto.SizeParam, utils.ToString(r.v))
	return queryParams
}

type ChatGetOptionWithStartsFromChatId struct {
	v int64
}

func NewChatGetOptionWithStartsFromChatId(v int64) *ChatGetOptionWithStartsFromChatId {
	return &ChatGetOptionWithStartsFromChatId{v: v}
}

func (r *ChatGetOptionWithStartsFromChatId) Apply(queryParams *url.Values) *url.Values {
	if queryParams == nil {
		queryParams = &url.Values{}
	}
	queryParams.Add(dto.ChatIdParam, utils.ToString(r.v))
	return queryParams
}

type ChatGetOptionWithStartsFromChatPinned struct {
	v bool
}

func NewChatGetOptionWithStartsFromChatPinned(v bool) *ChatGetOptionWithStartsFromChatPinned {
	return &ChatGetOptionWithStartsFromChatPinned{v: v}
}

func (r *ChatGetOptionWithStartsFromChatPinned) Apply(queryParams *url.Values) *url.Values {
	if queryParams == nil {
		queryParams = &url.Values{}
	}
	queryParams.Add(dto.PinnedParam, utils.ToString(r.v))
	return queryParams
}

type ChatGetOptionWithStartsFromChatLastUpdateDateTime struct {
	lastLastUpdateDateTime *time.Time
}

func NewChatGetOptionWithStartsFromChatLastUpdateDateTime(v *time.Time) *ChatGetOptionWithStartsFromChatLastUpdateDateTime {
	return &ChatGetOptionWithStartsFromChatLastUpdateDateTime{lastLastUpdateDateTime: v}
}

func (r *ChatGetOptionWithStartsFromChatLastUpdateDateTime) Apply(queryParams *url.Values) *url.Values {
	if queryParams == nil {
		queryParams = &url.Values{}
	}
	queryParams.Add(dto.LastUpdateDateTimeParam, r.lastLastUpdateDateTime.Format(time.RFC3339Nano))
	return queryParams
}

type ChatGetOptionWithSearch struct {
	s string
}

func NewChatGetOptionWithSearch(s string) *ChatGetOptionWithSearch {
	return &ChatGetOptionWithSearch{s: s}
}

func (r *ChatGetOptionWithSearch) Apply(queryParams *url.Values) *url.Values {
	if queryParams == nil {
		queryParams = &url.Values{}
	}
	queryParams.Add(dto.SearchStringParam, r.s)
	return queryParams
}

func (rc *TestRestClient) GetChats(ctx context.Context, behalfUserId int64, chatGetOptions ...ChatGetOption) ([]dto.ChatViewEnrichedDto, bool, error) {
	var queryParams *url.Values
	for _, opt := range chatGetOptions {
		if opt != nil {
			queryParams = opt.Apply(queryParams)
		}
	}

	res, err := query[any, dto.GetChatsResponseDto](ctx, &rc.restClient, behalfUserId, http.MethodGet, "/api/chat/search", "chat.Search", nil, queryParams)
	if err != nil {
		return []dto.ChatViewEnrichedDto{}, false, err
	}
	return res.Items, res.HasNext, nil
}

func (rc *TestRestClient) GetHasUnreadMessages(ctx context.Context, behalfUserId int64) (bool, error) {
	resp, err := query[any, dto.HasUnreadMessages](ctx, &rc.restClient, behalfUserId, http.MethodGet, "/api/chat/has-new-messages", "chat.HasUnreadMessages", nil, nil)
	if err != nil {
		return false, err
	}
	return resp.HasUnreadMessages, nil
}

func (rc *TestRestClient) GetReadMessageUsers(ctx context.Context, behalfUserId, chatId, messageId int64) (*dto.MessageReadResponse, error) {
	resp, err := query[any, dto.MessageReadResponse](ctx, &rc.restClient, behalfUserId, http.MethodGet, fmt.Sprintf("/api/chat/%d/message/read/%d", chatId, messageId), "message.ReadUsers", nil, nil)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (rc *TestRestClient) PutUserChatNotificationSettings(ctx context.Context, behalfUserId, chatId int64, consider bool) error {
	req := dto.PutChatNotificationSettingsDto{
		ConsiderMessagesOfThisChatAsUnread: consider,
	}
	return queryNoResponse[dto.PutChatNotificationSettingsDto](ctx, &rc.restClient, behalfUserId, http.MethodPut, "/api/chat/"+utils.ToString(chatId)+"/notification", "chat.PutUserChatNotificationSettings", &req, nil)
}

func (rc *TestRestClient) SearchBlogs(ctx context.Context) (dto.BlogPostsDTO, error) {
	return query[any, dto.BlogPostsDTO](ctx, &rc.restClient, dto.NonExistentUser, http.MethodGet, "/api/blog", "blog.Search", nil, nil)
}

type MessageCreateOption interface {
	Apply(*dto.MessageCreateDto)
}

type MessageCreateOptionResend struct {
	fromChatId int64
	messageId  int64
}

type MessageCreateOptionReply struct {
	messageId int64
}

func NewMessageCreateOptionResend(fromChatId, messageId int64) *MessageCreateOptionResend {
	return &MessageCreateOptionResend{
		fromChatId: fromChatId,
		messageId:  messageId,
	}
}

func NewMessageCreateOptionReply(messageId int64) *MessageCreateOptionReply {
	return &MessageCreateOptionReply{
		messageId: messageId,
	}
}

func (r *MessageCreateOptionResend) Apply(d *dto.MessageCreateDto) {
	d.EmbedMessageRequest = &dto.EmbedMessageRequest{
		Id:        r.messageId,
		ChatId:    r.fromChatId,
		EmbedType: dto.EmbedMessageTypeResend,
	}
}

func (r *MessageCreateOptionReply) Apply(d *dto.MessageCreateDto) {
	d.EmbedMessageRequest = &dto.EmbedMessageRequest{
		Id:        r.messageId,
		EmbedType: dto.EmbedMessageTypeReply,
	}
}

func (rc *TestRestClient) CreateMessage(ctx context.Context, behalfUserId int64, chatId int64, text string, messageCreateOptions ...MessageCreateOption) (int64, error) {
	req := dto.MessageCreateDto{
		Content: text,
	}

	for _, opt := range messageCreateOptions {
		if opt != nil {
			opt.Apply(&req)
		}
	}

	resp, err := query[dto.MessageCreateDto, dto.IdResponse](ctx, &rc.restClient, behalfUserId, http.MethodPost, "/api/chat/"+utils.ToString(chatId)+"/message", "message.Create", &req, nil)
	if err != nil {
		return 0, err
	}
	return resp.Id, nil
}

func (rc *TestRestClient) Reaction(ctx context.Context, behalfUserId int64, chatId, messageId int64, reaction string) error {
	req := dto.ReactionPutDto{
		Reaction: reaction,
	}

	return queryNoResponse[dto.ReactionPutDto](ctx, &rc.restClient, behalfUserId, http.MethodPut, "/api/chat/"+utils.ToString(chatId)+"/message/"+utils.ToString(messageId)+"/reaction", "message.Reaction", &req, nil)
}

func (rc *TestRestClient) EditMessage(ctx context.Context, behalfUserId int64, chatId, messageId int64, text string, messageCreateOptions ...MessageCreateOption) error {
	req := dto.MessageEditDto{
		Id: messageId,
		MessageCreateDto: dto.MessageCreateDto{
			Content: text,
		},
	}
	for _, opt := range messageCreateOptions {
		if opt != nil {
			opt.Apply(&req.MessageCreateDto)
		}
	}

	return queryNoResponse[dto.MessageEditDto](ctx, &rc.restClient, behalfUserId, http.MethodPut, "/api/chat/"+utils.ToString(chatId)+"/message", "message.Edit", &req, nil)
}

func (rc *TestRestClient) SyncMessage(ctx context.Context, behalfUserId int64, chatId, messageId int64) error {
	return queryNoResponse[any](ctx, &rc.restClient, behalfUserId, http.MethodPut, "/api/chat/"+utils.ToString(chatId)+"/message/"+utils.ToString(messageId)+"/sync-embed", "message.Sync", nil, nil)
}

func (rc *TestRestClient) DeleteMessage(ctx context.Context, behalfUserId int64, chatId, messageId int64) error {
	return queryNoResponse[any](ctx, &rc.restClient, behalfUserId, http.MethodDelete, "/api/chat/"+utils.ToString(chatId)+"/message/"+utils.ToString(messageId), "message.Delete", nil, nil)
}

func (rc *TestRestClient) PinMessage(ctx context.Context, behalfUserId int64, chatId, messageId int64, pin bool) error {
	var queryParams *url.Values = &url.Values{}
	queryParams.Set(dto.PinParam, utils.ToString(pin))

	return queryNoResponse[dto.MessageEditDto](ctx, &rc.restClient, behalfUserId, http.MethodPut, "/api/chat/"+utils.ToString(chatId)+"/message/"+utils.ToString(messageId)+"/pin", "message.Pin", nil, queryParams)
}

func (rc *TestRestClient) PublishMessage(ctx context.Context, behalfUserId int64, chatId, messageId int64, publish bool) error {
	var queryParams *url.Values = &url.Values{}
	queryParams.Set(dto.PublishParam, utils.ToString(publish))

	return queryNoResponse[dto.MessageEditDto](ctx, &rc.restClient, behalfUserId, http.MethodPut, "/api/chat/"+utils.ToString(chatId)+"/message/"+utils.ToString(messageId)+"/publish", "message.Publish", nil, queryParams)
}

type MessagePinnedGetOption interface {
	Apply(queryParams *url.Values) *url.Values
}

type MessagePinnedGetOptionWithSize struct {
	v int32
}

func NewMessagePinnedGetOptionWithSize(v int32) *MessagePinnedGetOptionWithSize {
	return &MessagePinnedGetOptionWithSize{v: v}
}

func (r *MessagePinnedGetOptionWithSize) Apply(queryParams *url.Values) *url.Values {
	if queryParams == nil {
		queryParams = &url.Values{}
	}
	queryParams.Add(dto.SizeParam, utils.ToString(r.v))
	return queryParams
}

func (rc *TestRestClient) GetPinnedMessages(ctx context.Context, behalfUserId int64, chatId int64, messageGetOptions ...MessagePinnedGetOption) ([]dto.PinnedMessageDto, error) {
	var queryParams *url.Values
	for _, opt := range messageGetOptions {
		if opt != nil {
			queryParams = opt.Apply(queryParams)
		}
	}

	res, err := query[any, dto.PinnedMessagesWrapper](ctx, &rc.restClient, behalfUserId, http.MethodGet, "/api/chat/"+utils.ToString(chatId)+"/message/pin", "message.Pinned", nil, queryParams)
	if err != nil {
		return []dto.PinnedMessageDto{}, err
	}
	return res.Data, nil
}

func (rc *TestRestClient) GetPinnedPromotedMessage(ctx context.Context, behalfUserId int64, chatId int64) (*dto.PinnedMessageDto, error) {
	res, err := query[any, *dto.PinnedMessageDto](ctx, &rc.restClient, behalfUserId, http.MethodGet, "/api/chat/"+utils.ToString(chatId)+"/message/pin/promoted", "message.PinnedPromoted", nil, nil)
	if err != nil {
		return nil, err
	}

	return res, nil
}

func (rc *TestRestClient) GetPublishedMessageForPublic(ctx context.Context, chatId, messageId int64) (*dto.MessageViewEnrichedDto, error) {
	res, err := query[any, *dto.PublishedMessageWrapper](ctx, &rc.restClient, dto.NonExistentUser, http.MethodGet, "/api/chat/public/"+utils.ToString(chatId)+"/message/"+utils.ToString(messageId), "message.PublishedPublic", nil, nil)
	if err != nil {
		return nil, err
	}

	if res == nil {
		return nil, nil
	}

	return res.Message, nil
}

func (rc *TestRestClient) GetPublishedMessages(ctx context.Context, behalfUserId int64, chatId int64) ([]dto.PublishedMessageDto, error) {
	res, err := query[any, dto.PublishedMessagesWrapper](ctx, &rc.restClient, behalfUserId, http.MethodGet, "/api/chat/"+utils.ToString(chatId)+"/message/publish", "message.Published", nil, nil)
	if err != nil {
		return nil, err
	}

	return res.Data, nil
}

type MessageGetOption interface {
	Apply(queryParams *url.Values) *url.Values
}

type MessageGetOptionWithSize struct {
	v int32
}

func NewMessageGetOptionWithSize(v int32) *MessageGetOptionWithSize {
	return &MessageGetOptionWithSize{v: v}
}

func (r *MessageGetOptionWithSize) Apply(queryParams *url.Values) *url.Values {
	if queryParams == nil {
		queryParams = &url.Values{}
	}
	queryParams.Add(dto.SizeParam, utils.ToString(r.v))
	return queryParams
}

type MessageGetOptionWithStartsFromItemId struct {
	v int64
}

func NewMessageGetOptionWithStartsFromItemId(v int64) *MessageGetOptionWithStartsFromItemId {
	return &MessageGetOptionWithStartsFromItemId{v: v}
}

func (r *MessageGetOptionWithStartsFromItemId) Apply(queryParams *url.Values) *url.Values {
	if queryParams == nil {
		queryParams = &url.Values{}
	}
	queryParams.Add(dto.StartingFromItemId, utils.ToString(r.v))
	return queryParams
}

type MessageGetOptionWithSearch struct {
	s string
}

func NewMessageGetOptionWithSearch(s string) *MessageGetOptionWithSearch {
	return &MessageGetOptionWithSearch{s: s}
}

func (r *MessageGetOptionWithSearch) Apply(queryParams *url.Values) *url.Values {
	if queryParams == nil {
		queryParams = &url.Values{}
	}
	queryParams.Add(dto.SearchStringParam, r.s)
	return queryParams
}

func (rc *TestRestClient) GetMessages(ctx context.Context, behalfUserId int64, chatId int64, messageGetOptions ...MessageGetOption) ([]dto.MessageViewEnrichedDto, bool, error) {
	var queryParams *url.Values
	for _, opt := range messageGetOptions {
		if opt != nil {
			queryParams = opt.Apply(queryParams)
		}
	}

	res, err := query[any, dto.MessagesResponseDto](ctx, &rc.restClient, behalfUserId, http.MethodGet, "/api/chat/"+utils.ToString(chatId)+"/message/search", "message.Search", nil, queryParams)
	if err != nil {
		return []dto.MessageViewEnrichedDto{}, false, err
	}
	return res.Items, res.HasNext, nil
}

func (rc *TestRestClient) MakeMessageBlogPost(ctx context.Context, behalfUserId int64, chatId, messageId int64) error {
	return queryNoResponse[any](ctx, &rc.restClient, behalfUserId, http.MethodPut, "/api/chat/"+utils.ToString(chatId)+"/message/"+utils.ToString(messageId)+"/blog-post", "message.MakeBlogPost", nil, nil)
}

func (rc *TestRestClient) SearchBlogComments(ctx context.Context, blogId int64) (dto.CommentsWrapper, error) {
	return query[any, dto.CommentsWrapper](ctx, &rc.restClient, dto.NonExistentUser, http.MethodGet, "/api/blog/"+utils.ToString(blogId)+"/comment", "blog.SearchComments", nil, nil)
}

// You must await after this command, because it takes a time to apply "ParticipantAdd" event
func (rc *TestRestClient) AddChatParticipants(ctx context.Context, behalfUserId int64, chatId int64, participantIds []int64) error {
	req := dto.ParticipantAddDto{
		ParticipantIds: participantIds,
	}
	return queryNoResponse[dto.ParticipantAddDto](ctx, &rc.restClient, behalfUserId, http.MethodPut, "/api/chat/"+utils.ToString(chatId)+"/participant", "participants.Add", &req, nil)
}

func (rc *TestRestClient) DeleteChatParticipants(ctx context.Context, behalfUserId int64, chatId int64, participantId int64) error {
	return queryNoResponse[any](ctx, &rc.restClient, behalfUserId, http.MethodDelete, "/api/chat/"+utils.ToString(chatId)+"/participant/"+utils.ToString(participantId), "participants.Delete", nil, nil)
}

func (rc *TestRestClient) ChangeChatParticipant(ctx context.Context, behalfUserId int64, chatId int64, participantId int64, newAdmin bool) error {
	query1 := url.Values{
		dto.AdminParam: []string{utils.ToString(newAdmin)},
	}
	return queryNoResponse[any](ctx, &rc.restClient, behalfUserId, http.MethodPut, "/api/chat/"+utils.ToString(chatId)+"/participant/"+utils.ToString(participantId), "participants.Change", nil, &query1)
}

func (rc *TestRestClient) LeaveChat(ctx context.Context, behalfUserId int64, chatId int64) error {
	return queryNoResponse[any](ctx, &rc.restClient, behalfUserId, http.MethodPut, "/api/chat/"+utils.ToString(chatId)+"/leave", "chat.Leave", nil, nil)
}

type ParticipantGetOption interface {
	Apply(queryParams *url.Values) *url.Values
}

type ParticipantGetOptionWithSearch struct {
	s string
}

func NewParticipantGetOptionWithSearch(s string) *ParticipantGetOptionWithSearch {
	return &ParticipantGetOptionWithSearch{s: s}
}

func (r *ParticipantGetOptionWithSearch) Apply(queryParams *url.Values) *url.Values {
	if queryParams == nil {
		queryParams = &url.Values{}
	}
	queryParams.Add(dto.SearchStringParam, r.s)
	return queryParams
}

func (rc *TestRestClient) GetChatParticipants(ctx context.Context, behalfUserId int64, chatId int64, participantGetOptions ...ParticipantGetOption) ([]*dto.UserViewEnrichedDto, int64, error) {
	var queryParams *url.Values
	for _, opt := range participantGetOptions {
		if opt != nil {
			queryParams = opt.Apply(queryParams)
		}
	}

	res, err := query[any, dto.ParticipantsWithAdminWrapper](ctx, &rc.restClient, behalfUserId, http.MethodGet, "/api/chat/"+utils.ToString(chatId)+"/participant/search", "participants.Get", nil, queryParams)
	if err != nil {
		return []*dto.UserViewEnrichedDto{}, 0, err
	}
	return res.Data, res.Count, nil
}

func (rc *TestRestClient) ReadMessage(ctx context.Context, behalfUserId int64, chatId, messageId int64) error {
	return queryNoResponse[any](ctx, &rc.restClient, behalfUserId, http.MethodPut, "/api/chat/"+utils.ToString(chatId)+"/message/read/"+utils.ToString(messageId), "message.Read", nil, nil)
}

func (rc *TestRestClient) MarkAllChatsAsRead(ctx context.Context, behalfUserId int64) error {
	return queryNoResponse[any](ctx, &rc.restClient, behalfUserId, http.MethodPut, "/api/chat/read", "message.ReadAllChats", nil, nil)
}

func (rc *TestRestClient) MarkChatAsRead(ctx context.Context, behalfUserId int64, chatId int64) error {
	return queryNoResponse[any](ctx, &rc.restClient, behalfUserId, http.MethodPut, "/api/chat/"+utils.ToString(chatId)+"/read", "message.ReadChat", nil, nil)
}

func (rc *TestRestClient) HealthCheck(ctx context.Context) error {
	return queryNoResponse[any](ctx, &rc.restClient, dto.NonExistentUser, http.MethodGet, "/internal/health", "internal.HealthCheck", nil, nil)
}
