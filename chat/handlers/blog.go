package handlers

import (
	"errors"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/labstack/echo/v4"
	"net/http"
	"nkonev.name/chat/auth"
	"nkonev.name/chat/db"
	. "nkonev.name/chat/logger"
	"nkonev.name/chat/services"
	"nkonev.name/chat/utils"
)

type BlogHandler struct {
	db          *db.DB
	notificator services.Events
	policy      *services.SanitizerPolicy
}

func NewBlogHandler(db *db.DB, notificator services.Events, policy *services.SanitizerPolicy) *BlogHandler {
	return &BlogHandler{
		db:          db,
		notificator: notificator,
		policy:      policy,
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
	Id      int64  `json:"id"`
	Title   string `json:"title"`
	Preview string `json:"preview"`
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

	// get chats where blog=true
	blogs, err := h.db.GetBlogPostsByLimitOffset(size, offset)
	if err != nil {
		return err
	}
	// get their message where blog_post=true for sake to make preview
	return c.JSON(http.StatusOK, blogs)
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
