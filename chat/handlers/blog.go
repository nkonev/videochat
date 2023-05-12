package handlers

import (
	"errors"
	"github.com/PuerkitoBio/goquery"
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
	"strings"
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

type BlogPostPreviewDto struct {
	Id             int64     `json:"id"` // chatId
	Title          string    `json:"title"`
	CreateDateTime time.Time `json:"createDateTime"`
	OwnerId        *int64    `json:"ownerId"`
	Owner          *dto.User `json:"owner"`
	MessageId      *int64    `json:"messageId"`
	Text           *string   `json:"-"`
	Preview        *string   `json:"preview"`
	ImageUrl       *string   `json:"imageUrl"`
}

func getSize(size int, isSearch bool) int {
	if isSearch {
		return viper.GetInt("blogSearchSize")
	} else {
		return size
	}
}

func getOffset(offset int, isSearch bool) int {
	if isSearch {
		return 0
	} else {
		return offset
	}
}

func (h *BlogHandler) GetBlogPosts(c echo.Context) error {

	page := utils.FixPageString(c.QueryParam("page"))
	size := utils.FixSizeString(c.QueryParam("size"))
	offset := utils.GetOffset(page, size)
	searchString := c.QueryParam("searchString")
	searchString = strings.TrimSpace(searchString)

	isSearch := false

	if len(searchString) != 0 {
		isSearch = true
	}

	return db.Transact(h.db, func(tx *db.Tx) error {
		// get chats where blog=true
		blogs, err := tx.GetBlogPostsByLimitOffset(getSize(size, isSearch), getOffset(offset, isSearch))
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
					blogPost.ImageUrl = h.tryGetFirstImage(post.Text)
					t := post.Text
					blogPost.Text = &t
					blogPost.Preview = h.cutText(post.Text)
					oid := post.OwnerId
					blogPost.OwnerId = &oid
					mid := post.MessageId
					blogPost.MessageId = &mid
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

		if isSearch {
			search, err := h.performSearchAndPaging(searchString, response, size, offset)
			if err != nil {
				return err
			}
			return c.JSON(http.StatusOK, search)
		} else {
			return c.JSON(http.StatusOK, response)
		}
	})
}

func (h *BlogHandler) performSearchAndPaging(searchString string, searchable []*BlogPostPreviewDto, size, offset int) ([]*BlogPostPreviewDto, error) {
	searchString = strings.ToLower(searchString)

	var intermediateList = make([]*BlogPostPreviewDto, 0)

	for _, blogPostPreviewDto := range searchable {
		if strings.Contains(strings.ToLower(blogPostPreviewDto.Title), searchString) ||
			(blogPostPreviewDto.Preview != nil && strings.Contains(strings.ToLower(*blogPostPreviewDto.Text), searchString)) {
			intermediateList = append(intermediateList, blogPostPreviewDto)
		}
	}

	var list = make([]*BlogPostPreviewDto, 0)
	var counter = 0
	var respCounter = 0

	for _, objInfo := range intermediateList {
		if counter >= offset {
			list = append(list, objInfo)
			respCounter++
			if respCounter >= size {
				break
			}
		}
		counter++
	}

	// get their message where blog_post=true for sake to make preview
	return list, nil
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

func (h *BlogHandler) tryGetFirstImage(text string) *string {
	// Load the HTML document
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(text))
	if err != nil {
		Logger.Warnf("Unagle to get image: %v", err)
		return nil
	}

	maybeImage := doc.Find("img").First()
	if maybeImage != nil {
		src, exists := maybeImage.Attr("src")
		if exists {
			return &src
		}
	}
	return nil
}

func (h *BlogHandler) cutText(text string) *string {
	tmp := h.stripTagsPolicy.Sanitize(text)
	runes := []rune(tmp)
	size := utils.Min(viper.GetInt("blogPreviewMaxTextSize"), len(runes))
	ret := string(runes[:size])
	return &ret
}