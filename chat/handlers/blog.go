package handlers

import (
	"errors"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
	"net/http"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/client"
	"nkonev.name/chat/db"
	"nkonev.name/chat/dto"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/services"
	"nkonev.name/chat/utils"
	"time"
)

type BlogHandler struct {
	db              *db.DB
	notificator     services.Events
	policy          *services.SanitizerPolicy
	stripTagsPolicy *services.StripTagsPolicy
	restClient      *client.RestClient
}

func NewBlogHandler(db *db.DB, notificator services.Events, policy *services.SanitizerPolicy, stripTagsPolicy *services.StripTagsPolicy, restClient *client.RestClient) *BlogHandler {
	return &BlogHandler{
		db:              db,
		notificator:     notificator,
		policy:          policy,
		stripTagsPolicy: stripTagsPolicy,
		restClient:      restClient,
	}
}

type CreateBlogDto struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

func (a *CreateBlogDto) Validate() error {
	return validation.ValidateStruct(a,
		validation.Field(&a.Title, validation.Required, validation.Length(minChatNameLen, maxChatNameLen), validation.NotIn(db.ReservedPublicallyAvailableForSearchChats)),
		validation.Field(&a.Text, validation.Required, validation.Length(minMessageLen, maxMessageLen)),
	)
}

func (h *BlogHandler) CreateBlogPost(c echo.Context) error {
	// auth check
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	var bindTo = new(CreateBlogDto)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	if valid, err := ValidateAndRespondError(c, bindTo); err != nil || !valid {
		return err
	}

	return db.Transact(h.db, func(tx *db.Tx) error {
		// db.CreateChat (blog=true)
		chatId, _, err := tx.CreateChat(&db.Chat{
			Title:             TrimAmdSanitize(h.policy, bindTo.Title),
			CanResend:         true,
			AvailableToSearch: true,
			Blog:              true,
		})
		if err != nil {
			return err
		}
		// db.InsertMessage
		_, _, _, err = tx.CreateMessage(&db.Message{
			Text:     TrimAmdSanitize(h.policy, bindTo.Text),
			ChatId:   chatId,
			OwnerId:  userPrincipalDto.UserId,
			BlogPost: true,
		})
		return err
	})
}

type BlogPostPreviewDto struct {
	Id             int64     `json:"id"` // chatId
	Title          string    `json:"title"`
	CreateDateTime time.Time `json:"createDateTime"`
	OwnerId        *int64    `json:"ownerId"`
	Owner          *dto.User `json:"owner"`
	MessageId      *int64    `json:"messageId"`
	Preview        *string   `json:"preview"`
}

func (h *BlogHandler) GetBlogPosts(c echo.Context) error {
	// auth check
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	page := utils.FixPageString(c.QueryParam("page"))
	size := utils.FixSizeString(c.QueryParam("size"))
	offset := utils.GetOffset(page, size)

	return db.Transact(h.db, func(tx *db.Tx) error {
		// get chats where blog=true
		blogs, err := tx.GetBlogPostsByLimitOffset(size, offset)
		if err != nil {
			return err
		}

		var blogIds []int64 = make([]int64, 0)
		for _, blog := range blogs {
			blogIds = append(blogIds, blog.Id)
		}
		posts, err := tx.BlogPosts(blogIds)
		if err != nil {
			return err
		}
		var response = make([]*BlogPostPreviewDto, 0)
		for _, blog := range blogs {

			blogPost := &BlogPostPreviewDto{
				Id:             blog.Id,
				CreateDateTime: blog.CreateDateTime,
				Title:          blog.Title,
			}

			for _, post := range posts {
				if post.ChatId == blog.Id {
					tmp := h.stripTagsPolicy.Sanitize(post.Text)
					max := viper.GetInt("blogPreviewMaxTextSize")
					tmp = tmp[:utils.Min(max, len(tmp))]
					blogPost.Preview = &tmp
					blogPost.OwnerId = &post.OwnerId
					blogPost.MessageId = &post.MessageId
					break
				}
			}

			response = append(response, blogPost)
		}

		var participantIdSet = map[int64]bool{}
		for _, respDto := range response {
			if respDto.OwnerId != nil {
				participantIdSet[*respDto.OwnerId] = true
			}
		}
		var users = getUsersRemotelyOrEmpty(participantIdSet, h.restClient, c)

		for _, respDto := range response {
			if respDto.OwnerId != nil {
				respDto.Owner = users[*respDto.OwnerId]
			}
		}

		// get their message where blog_post=true for sake to make preview
		return c.JSON(http.StatusOK, response)
	})
}

//	func (h *BlogHandler) GetComments(c echo.Context) error {
//		// auth check
//		// get messages where blog_post=false for sake to make preview
//	}
type RenameBlogPost struct {
	Id    int64  `json:"id"`
	Title string `json:"title"`
}

func (a *RenameBlogPost) Validate() error {
	return validation.ValidateStruct(a,
		validation.Field(&a.Title, validation.Required, validation.Length(minChatNameLen, maxChatNameLen), validation.NotIn(db.ReservedPublicallyAvailableForSearchChats)),
	)
}

func (h *BlogHandler) RenameBlogPost(c echo.Context) error {
	// auth check
	var userPrincipalDto, ok = c.Get(utils.USER_PRINCIPAL_DTO).(*auth.AuthResult)
	if !ok || userPrincipalDto == nil {
		GetLogEntry(c.Request().Context()).Errorf("Error during getting auth context")
		return errors.New("Error during getting auth context")
	}

	var bindTo = new(RenameBlogPost)
	if err := c.Bind(bindTo); err != nil {
		GetLogEntry(c.Request().Context()).Warnf("Error during binding to dto %v", err)
		return err
	}

	if valid, err := ValidateAndRespondError(c, bindTo); err != nil || !valid {
		return err
	}

	return db.Transact(h.db, func(tx *db.Tx) error {
		err := tx.RenameChat(bindTo.Id, bindTo.Title)
		return err
	})

}
